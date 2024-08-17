package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync"

	_ "github.com/go-sql-driver/mysql"
)

type Message struct {
	ID        int
	Content   string
	CreatedAt string
}

var (
	db   *sql.DB
	mu   sync.Mutex
	tmpl *template.Template
)

func main() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Initialize database connection
	var err error
	db, err = sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/golang_webapp") // Update UserName and Password
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/submit", submitHandler)
	//http.HandleFunc("/submitRecommend", submitRecommendedHandler)
	http.HandleFunc("/recommend", getRecommendedHandler)

	fmt.Println("Starting server on :8080...")
	http.ListenAndServe(":8080", nil)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	// Retrieve messages from the database
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	mu.Lock()
	rows, err := db.Query("SELECT id, content, created_at FROM messages ORDER BY created_at DESC")
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
    JOIN favorites f ON m.id = f.message_id
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
