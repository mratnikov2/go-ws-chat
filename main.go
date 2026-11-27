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
	ID     int64  `json:"id,omitempty"`
	Type   string `json:"type"`
	Author string `json:"author"`
	Text   string `json:"text"`
	Time   string `json:"time"`
	IsRead bool   `json:"is_read"`
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
		time TEXT,
		is_read INTEGER DEFAULT 0
	)`)
	if err != nil {
		log.Fatal(err)
	}

	// Добавляем колонку is_read в уже существующую таблицу (ошибку игнорируем, если колонка уже есть)
	db.Exec(`ALTER TABLE messages ADD COLUMN is_read INTEGER DEFAULT 0`)

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

		// 1. Отправляем историю с учётом статуса прочтения
		rows, _ := db.Query("SELECT id, author, content, time, COALESCE(is_read, 0) FROM messages ORDER BY id ASC")
		for rows.Next() {
			var msg ChatMessage
			var isReadInt int
			msg.Type = "chat"
			rows.Scan(&msg.ID, &msg.Author, &msg.Text, &msg.Time, &isReadInt)
			msg.IsRead = isReadInt == 1
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

			// 3. Уведомляем о выходе
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

			// Обработка события прочтения
			if msg.Type == "read" {
				res, err := db.Exec("UPDATE messages SET is_read = 1 WHERE author != ? AND is_read = 0", name)
				if err == nil {
					affected, _ := res.RowsAffected()
					if affected > 0 {
						broadcast(ChatMessage{
							Type:   "read",
							Author: name,
						})
					}
				}
				continue
			}

			// Обычное сообщение
			msg.Type = "chat"
			msg.Author = name
			msg.Time = time.Now().Format("15:04")
			msg.IsRead = false

			res, err := db.Exec("INSERT INTO messages (author, content, time, is_read) VALUES (?, ?, ?, 0)", msg.Author, msg.Text, msg.Time)
			if err == nil {
				msg.ID, _ = res.LastInsertId()
			}

			broadcast(msg)
		}
	})

	fmt.Println("Сервер запущен на http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
