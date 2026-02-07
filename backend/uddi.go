package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var db *sql.DB

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Home Page")
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func reader(conn *websocket.Conn) {
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}

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

		_, err = db.Exec(
			`INSERT INTO events (event_type, message_id, user_id, channel_id, created_at)
			 VALUES ($1, $2, $3, $4, NOW())`,
			"message_sent", messageID, "anonymous", "general",
		)
		if err != nil {
			log.Fatal(err)
		}

		if err := conn.WriteMessage(messageType, p); err != nil {
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

	fmt.Println("Server started on :8080")

	log.Fatal(http.ListenAndServe(":8080", nil))
}
