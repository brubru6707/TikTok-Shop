package main

import (
	"fmt"
	"html/template"
	"net/http"
	"sync"
)

type Message struct {
	Content string
}

var (
	messages []Message
	mu       sync.Mutex
)

func main() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/submit", submitHandler)

	fmt.Println("Starting server on :8080...")
	http.ListenAndServe(":8080", nil)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, "Error parsing template", http.StatusInternalServerError)
		return
	}

	mu.Lock()
	defer mu.Unlock()

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

	mu.Lock()
	messages = append(messages, Message{Content: content})
	mu.Unlock()

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
