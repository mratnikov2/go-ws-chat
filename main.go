package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	_ "modernc.org/sqlite"
)

type ChatMessage struct {
	Type   string `json:"type"`
	Author string `json:"author"`
	Text   string `json:"text"`
	Time   string `json:"time"`
}

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
var clients = make(map[*websocket.Conn]string)
var mu sync.Mutex
var db *sql.DB

const channelPassword = "123"
const maxClients = 2

func broadcast(msg ChatMessage) {
	mu.Lock()
	defer mu.Unlock()
	for client := range clients {
		client.WriteJSON(msg)
	}
}

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
		pass := r.URL.Query().Get("pass")
		if pass != channelPassword {
			http.Error(w, "Неверный пароль", http.StatusUnauthorized)
			return
		}

		name := r.URL.Query().Get("name")
		if name == "" {
			name = "Аноним"
		}

		mu.Lock()
		if len(clients) >= maxClients {
			mu.Unlock()
			http.Error(w, "Комната заполнена", http.StatusForbidden)
			return
		}
		mu.Unlock()

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		mu.Lock()
		if len(clients) >= maxClients {
			mu.Unlock()
			conn.Close()
			return
		}
		clients[conn] = name
		mu.Unlock()

		// 1. Сначала отправляем новичку историю
		rows, _ := db.Query("SELECT author, content, time FROM messages ORDER BY id ASC")
		for rows.Next() {
			var msg ChatMessage
			msg.Type = "chat"
			rows.Scan(&msg.Author, &msg.Text, &msg.Time)
			conn.WriteJSON(msg)
		}
		rows.Close()

		// 2. Уведомляем всех о входе
		broadcast(ChatMessage{
			Type: "system",
			Text: name + " в чате",
			Time: time.Now().Format("15:04"),
		})

		defer func() {
			mu.Lock()
			delete(clients, conn)
			mu.Unlock()
			conn.Close()

			// 3. Уведомляем оставшихся о выходе
			broadcast(ChatMessage{
				Type: "system",
				Text: name + " вышел(ла) из чата",
				Time: time.Now().Format("15:04"),
			})
		}()

		for {
			var msg ChatMessage
			err := conn.ReadJSON(&msg)
			if err != nil {
				break
			}

			msg.Type = "chat"
			msg.Time = time.Now().Format("15:04")
			db.Exec("INSERT INTO messages (author, content, time) VALUES (?, ?, ?)", msg.Author, msg.Text, msg.Time)

			broadcast(msg)
		}
	})

	fmt.Println("Сервер запущен на http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
