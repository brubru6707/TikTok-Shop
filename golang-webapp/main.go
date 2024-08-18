package main

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/websocket"
)

var (
	db   *sql.DB
	tmpl *template.Template
	rdb  *redis.Client
	ctx  = context.Background()
)

type Message struct {
	ID        int
	Content   string
	CreatedAt string
}

type Favorite struct {
	ID        int
	Content   string
	CreatedAt string
}

type Response struct {
	Favorites []Favorite
	Messages  []Message
}

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func main() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	var err error
	db, err = sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/golang_webapp")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rdb = redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer rdb.Close()

	tmpl = template.Must(template.ParseFiles("static/templates/index.html"))

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/submit", submitHandler)
	http.HandleFunc("/delete", deleteHandler)
	http.HandleFunc("/notifications", notificationHandler)
	http.HandleFunc("/submitRecommend", submitRecommendHandler)
	http.HandleFunc("/recommend", getRecommendedHandler)
	http.HandleFunc("/deleteFavorite", deleteFavoriteHandler)

	fmt.Println("Starting server on :8080...")
	http.ListenAndServe(":8080", nil)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, content, created_at FROM messages ORDER BY created_at DESC")
	if err != nil {
		http.Error(w, "Error retrieving messages", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.ID, &msg.Content, &msg.CreatedAt); err != nil {
			http.Error(w, "Error scanning message", http.StatusInternalServerError)
			return
		}
		messages = append(messages, msg)
	}

	fav_rows, err := db.Query(`
		SELECT m.id, m.content, m.created_at
		FROM messages m
		JOIN favorites as f ON m.id = f.message_id
	`)
	if err != nil {
		http.Error(w, "Error retrieving messages", http.StatusInternalServerError)
		return
	}
	defer fav_rows.Close()

	var favorites []Favorite
	for fav_rows.Next() {
		var msg Favorite
		if err := fav_rows.Scan(&msg.ID, &msg.Content, &msg.CreatedAt); err != nil {
			http.Error(w, "Error scanning message", http.StatusInternalServerError)
			return
		}
		favorites = append(favorites, msg)
	}
	response := Response{
		Favorites: favorites,
		Messages:  messages,
	}

	tmpl.Execute(w, response)
}

func submitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	content := r.FormValue("content")
	if content == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	_, err := db.Exec("INSERT INTO messages (content) VALUES (?)", content)
	if err != nil {
		http.Error(w, "Error saving message", http.StatusInternalServerError)
		return
	}

	if err := rdb.Publish(ctx, "notifications", content).Err(); err != nil {
		log.Printf("Error publishing to Redis: %v", err)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func notificationHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade failed:", err)
		return
	}
	defer conn.Close()

	pubsub := rdb.Subscribe(ctx, "notifications")
	defer pubsub.Close()

	go func() {
		for {
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println("Error sending ping:", err)
				return
			}
			time.Sleep(30 * time.Second)
		}
	}()

	for {
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			log.Println("Error receiving message from Redis:", err)
			return
		}
		if err := conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
			log.Println("Error writing to WebSocket:", err)
			return
		}
	}
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "ID parameter is required", http.StatusBadRequest)
		return
	}

	// Delete from favorites
	_, err := db.Exec("DELETE FROM favorites WHERE message_id = ?", id)
	if err != nil {
		http.Error(w, "Failed to delete from favorites", http.StatusInternalServerError)
		return
	}

	// Delete from messages
	_, err = db.Exec("DELETE FROM messages WHERE id = ?", id)
	if err != nil {
		http.Error(w, "Error deleting message", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func submitRecommendHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "ID parameter is required", http.StatusBadRequest)
		return
	}

	result, err := db.Exec("INSERT INTO favorites (message_id) VALUES (?)", id)
	if err != nil {
		http.Error(w, "Error inserting message", http.StatusInternalServerError)
		return
	}

	if rowsAffected, _ := result.RowsAffected(); rowsAffected == 0 {
		http.Error(w, "No message found with the given ID", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func deleteFavoriteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "ID parameter is required", http.StatusBadRequest)
		return
	}

	// Delete from favorites
	_, err := db.Exec("DELETE FROM favorites WHERE message_id = ?", id)
	if err != nil {
		http.Error(w, "Failed to delete from favorites", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func getRecommendedHandler(w http.ResponseWriter, r *http.Request) {

}
