package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

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

		fmt.Println(string(p))

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
	fs := http.FileServer(http.Dir("../web"))
	http.Handle("/", fs)

	http.HandleFunc("/ws", websocketEndpoint)

	fmt.Println("Server started on :8080")

	log.Fatal(http.ListenAndServe(":8080", nil))
}
