package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type OutgoingMessage struct {
	Type      string `json:"type"`
	EventID   int64  `json:"event_id"`
	Content   string `json:"content"`
	UserID    string `json:"user_id"`
	ChannelID string `json:"channel_id"`
}

type IncomingSync struct {
	Type        string `json:"type"`
	LastEventID int64  `json:"last_event_id"`
}

var db *sql.DB

var (
	clients   = make(map[*websocket.Conn]bool)
	clientsMu sync.Mutex
)

func broadcast(data []byte) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	for conn := range clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Println("Broadcast write failed:", err)
			conn.Close()
			delete(clients, conn)
		}
	}
}

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Home Page")
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func reader(conn *websocket.Conn) {
	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}

		// Try to parse as JSON sync request
		var incoming IncomingSync
		if json.Unmarshal(p, &incoming) == nil && incoming.Type == "sync" {
			handleSync(conn, incoming.LastEventID)
			continue
		}

		// Otherwise treat as plain chat message
		content := string(p)
		fmt.Println(content)

		var messageID int64
		err = db.QueryRow(
			`INSERT INTO messages (user_id, channel_id, content, platform, created_at)
			 VALUES ($1, $2, $3, $4, NOW())
			 RETURNING id`,
			"anonymous", "general", content, "twitch",
		).Scan(&messageID)
		if err != nil {
			log.Fatal(err)
		}

		var eventID int64
		err = db.QueryRow(
			`INSERT INTO events (event_type, message_id, user_id, channel_id, created_at)
			 VALUES ($1, $2, $3, $4, NOW())
			 RETURNING id`,
			"message_sent", messageID, "anonymous", "general",
		).Scan(&eventID)
		if err != nil {
			log.Fatal(err)
		}

		reply := OutgoingMessage{
			Type:      "message",
			EventID:   eventID,
			Content:   content,
			UserID:    "anonymous",
			ChannelID: "general",
		}
		replyJSON, err := json.Marshal(reply)
		if err != nil {
			log.Println("Failed to marshal reply:", err)
			return
		}
		broadcast(replyJSON)
	}
}

func handleSync(conn *websocket.Conn, lastEventID int64) {
	rows, err := db.Query(
		`SELECT e.id, m.content, m.user_id, m.channel_id
		 FROM events e JOIN messages m ON e.message_id = m.id
		 WHERE e.id > $1 ORDER BY e.id ASC`,
		lastEventID,
	)
	if err != nil {
		log.Println("Sync query failed:", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var msg OutgoingMessage
		if err := rows.Scan(&msg.EventID, &msg.Content, &msg.UserID, &msg.ChannelID); err != nil {
			log.Println("Sync row scan failed:", err)
			return
		}
		msg.Type = "message"
		data, err := json.Marshal(msg)
		if err != nil {
			log.Println("Sync marshal failed:", err)
			return
		}
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Println(err)
			return
		}
	}
}

func websocketEndpoint(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	log.Println("Client Connected")

	clientsMu.Lock()
	clients[ws] = true
	clientsMu.Unlock()

	defer func() {
		clientsMu.Lock()
		delete(clients, ws)
		clientsMu.Unlock()
		ws.Close()
	}()

	err = ws.WriteMessage(websocket.TextMessage, []byte("Welcome to the UDDI server!"))
	if err != nil {
		log.Println(err)
		return
	}

	reader(ws)
}

func main() {
	godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	var err error
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	log.Println("Connected to database")

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS messages (
			id BIGSERIAL PRIMARY KEY,
			user_id TEXT NOT NULL,
			channel_id TEXT NOT NULL,
			content TEXT NOT NULL,
			platform TEXT,
			reply_to_id BIGINT,
			edited_at TIMESTAMP,
			deleted_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Fatal("Failed to create MESSAGES table:", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS events (
			id BIGSERIAL PRIMARY KEY,
			event_type TEXT NOT NULL,
			message_id BIGINT,
			user_id TEXT,
			channel_id TEXT,
			data JSONB,
			created_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Fatal("Failed to create EVENTS table:", err)
	}

	fs := http.FileServer(http.Dir("../web"))
	http.Handle("/", fs)

	http.HandleFunc("/ws", websocketEndpoint)

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

	fmt.Println("Server started on :" + port)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
