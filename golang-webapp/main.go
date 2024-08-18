package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/websocket"
)

var (
	db   *sql.DB
	mu   sync.Mutex
	tmpl *template.Template
	rdb  *redis.Client
	ctx  context.Context
)

type Message struct {
	ID        int
	Content   string
	CreatedAt string
}

type requestData struct {
	MsgID string `json:"msg_id"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func main() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	ctx = context.Background()

	// Initialize Redis client
	rdb = redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // Redis server address
	})
	defer rdb.Close()

	// Initialize database connection
	var err error
	db, err = sql.Open("mysql", "root:password@tcp(127.0.0.1:3306)/tiktok_db") // Update UserName and Password
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Load the HTML template
	tmpl = template.Must(template.ParseFiles("static/templates/index.html"))

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/submit", submitHandler)
	http.HandleFunc("/delete", deleteHandler)
	http.HandleFunc("/notifications", notificationHandler)
	http.HandleFunc("/submitRecommend", submitRecommendHandler)
	http.HandleFunc("/recommend", getRecommendedHandler)
	http.HandleFunc("/deleteFavorite", deleteFavoriteHandler) // New handler for deleting from favorites

	fmt.Println("Starting server on :8080...")
	http.ListenAndServe(":8080", nil)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	rows, err := db.Query("SELECT id, content, created_at FROM messages ORDER BY created_at DESC")
	mu.Unlock()

	if err != nil {
		log.Println("Error fetching messages:", err)
		http.Error(w, "Error retrieving messages", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		err := rows.Scan(&msg.ID, &msg.Content, &msg.CreatedAt)
		if err != nil {
			http.Error(w, "Error scanning message", http.StatusInternalServerError)
			return
		}
		messages = append(messages, msg)
	}

	tmpl.Execute(w, messages)
}

func submitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	content := r.FormValue("content")
	if content == "" {
		http.Error(w, "Content cannot be empty", http.StatusBadRequest)
		return
	}

	// Insert the new message into the database
	mu.Lock()
	_, err := db.Exec("INSERT INTO messages (content) VALUES (?)", content)
	mu.Unlock()

	if err != nil {
		http.Error(w, "Error saving message", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)

	if rdb == nil {
		http.Error(w, "Redis client not initialized", http.StatusInternalServerError)
		return
	}

	// Publish notification to Redis
	err = rdb.Publish(ctx, "notifications", content).Err()
	if err != nil {
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

	ticker := time.NewTicker(time.Second * 30) // Ping every 30 seconds
	defer ticker.Stop()

	go func() {
		for {
			<-ticker.C
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println("Error sending ping:", err)
				return
			}
		}
	}()

	// Listen for messages from Redis and send them to the WebSocket client
	for {
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			log.Println("Error receiving message from Redis:", err)
			return
		}

		log.Println("Received message from Redis:", msg.Payload)

		err = conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload))
		if err != nil {
			log.Println("Error writing to WebSocket:", err)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket closed unexpectedly: %v", err)
			}
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
	log.Println("Received request to delete message with ID:", id) // Log received ID
	if id == "" {
		http.Error(w, "ID parameter is required", http.StatusBadRequest)
		return
	}

	// Delete the message from the database
	mu.Lock()
	result, err := db.Exec("DELETE FROM messages WHERE id = ?", id)
	mu.Unlock()

	if err != nil {
		log.Println("Error deleting message:", err) // Log deletion error
		http.Error(w, "Error deleting message", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Println("Error checking deletion result:", err) // Log error when checking rows affected
		http.Error(w, "Error checking deletion result", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		log.Println("No message found with the given ID:", id) // Log if no rows were affected
		http.Error(w, "No message found with the given ID", http.StatusNotFound)
		return
	}

	log.Println("Message with ID:", id, "deleted successfully.") // Confirm deletion
	w.WriteHeader(http.StatusOK)
}

func getRecommendedHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("recommend.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mu.Lock()
	rows, err := db.Query(`
		SELECT m.id, m.content, m.created_at
		FROM messages m
		JOIN favorites as f ON m.id = f.message_id
	`)
	mu.Unlock()

	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving messages: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		err := rows.Scan(&msg.ID, &msg.Content, &msg.CreatedAt)
		if err != nil {
			http.Error(w, "Error scanning message", http.StatusInternalServerError)
			return
		}
		messages = append(messages, msg)
	}

	html_err := tmpl.Execute(w, messages)
	if html_err != nil {
		http.Error(w, "Error rendering recommended webpage", http.StatusInternalServerError)
		return
	}
}

func deleteMessageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	// First, delete all favorites associated with the message
	_, err := db.Exec("DELETE FROM favorites WHERE message_id = ?", id)
	if err != nil {
		log.Printf("Error deleting favorites for message ID %s: %v", id, err)
		http.Error(w, "Failed to delete favorites", http.StatusInternalServerError)
		return
	}

	// Then, delete the message itself
	_, err = db.Exec("DELETE FROM messages WHERE id = ?", id)
	if err != nil {
		log.Printf("Error deleting message with ID %s: %v", id, err)
		http.Error(w, "Failed to delete message", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func submitRecommendHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		MsgID string `json:"msg_id"`
	}

	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Log the received favorite request for debugging
	log.Printf("Received favorite for message ID: %s", data.MsgID)

	// Return a success response without necessarily saving to a database
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"success": true}`))
}

func deleteFavoriteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	// Logic to delete the favorite from the database or data store
	// Example:
	// _, err := db.Exec("DELETE FROM favorites WHERE id = ?", id)

	w.WriteHeader(http.StatusOK)
}
