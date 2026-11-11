package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	_ "modernc.org/sqlite"
)

type ChatMessage struct {
	Author string `json:"author"`
	Text   string `json:"text"`
	Time   string `json:"time"`
}

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
var clients = make(map[*websocket.Conn]bool)
var db *sql.DB

// Наш секретный пароль для входа в канал
const channelPassword = "123"

func main() {
	var err error
	db, err = sql.Open("sqlite", "chat.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT, 
		author TEXT,
		content TEXT,
		time TEXT
	)`)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		// 1. Проверяем пароль ДО того, как пустить человека в чат
		pass := r.URL.Query().Get("pass")
		if pass != channelPassword {
			http.Error(w, "Неверный пароль", http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		clients[conn] = true

		rows, _ := db.Query("SELECT author, content, time FROM messages ORDER BY id ASC")
		for rows.Next() {
			var msg ChatMessage
			rows.Scan(&msg.Author, &msg.Text, &msg.Time)
			conn.WriteJSON(msg)
		}
		rows.Close()

		for {
			var msg ChatMessage
			err := conn.ReadJSON(&msg)
			if err != nil {
				delete(clients, conn)
				break
			}

			msg.Time = time.Now().Format("15:04")
			db.Exec("INSERT INTO messages (author, content, time) VALUES (?, ?, ?)", msg.Author, msg.Text, msg.Time)

			for client := range clients {
				client.WriteJSON(msg)
			}
		}
	})

	fmt.Println("Сервер запущен на http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
