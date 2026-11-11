# Private Messenger (Prototype)

A secure personal communication channel designed for real-time messaging without surveillance, ads, or phone number binding.

## 🎯 Goals and Users
* **Author:** creates, edits, and deletes messages; manages the channel.
* **Reader:** instantly receives new messages and reads the history.

## 🛠 Tech Stack
* **Frontend:** Vue.js 3 + Vite + Tailwind CSS (PWA)
* **Backend:** Go (Golang) — `net/http` and `gorilla/websocket`
* **Database:** SQLite (with future migration to PostgreSQL via the `pgx` driver)
* **Deployment:** Single binary + Podman (planned)

## ✨ Current Features (MVP)
* Closed access via a secret password/key.
* Two-way communication and instant message delivery via WebSockets.
* Reliable chat history storage in a relational database.
* Automatic loading of old messages upon entry.
* Delivery and read statuses (in development).

## 🚀 Roadmap
1. Transition from a "Channel" format to a full-fledged "Chat" (ability to reply and leave reactions).
2. Media file support (images, documents, voice messages).
3. Implementation of End-to-End (E2E) encryption on the client side.
4. Development of a fully-featured PWA application for smartphone installation.

## 💻 Development Setup
1. Install project dependencies: `go mod tidy`
2. Run the local server: `go run main.go`
3. Use tunneling (e.g., Pinggy) for external access:
   `ssh -p 443 -R0:localhost:8080 a.pinggy.io`
   