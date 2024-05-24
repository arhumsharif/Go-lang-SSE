package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type SSEClient struct {
	writer  http.ResponseWriter
	flusher http.Flusher
}

var clients = struct {
	sync.RWMutex
	list []*SSEClient
}{}

func sseHandler(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	client := &SSEClient{writer: w, flusher: flusher}

	// Add the new client to the clients list
	clients.Lock()
	clients.list = append(clients.list, client)
	clients.Unlock()

	// Keep the connection open
	notify := w.(http.CloseNotifier).CloseNotify()
	<-notify

	// Remove the client when connection is closed
	clients.Lock()
	for i, c := range clients.list {
		if c == client {
			clients.list = append(clients.list[:i], clients.list[i+1:]...)
			break
		}
	}
	clients.Unlock()
}

func triggerUpdateHandler(w http.ResponseWriter, r *http.Request) {
	// Broadcast a message to all connected clients
	message := fmt.Sprintf("data: Update triggered at %s\n\n", time.Now().Format(time.RFC3339))

	clients.RLock()
	for _, client := range clients.list {
		client.writer.Write([]byte(message))
		client.flusher.Flush()
	}
	clients.RUnlock()

	w.Write([]byte("Update triggered\n"))
}

func main() {
	http.HandleFunc("/events", sseHandler)
	http.HandleFunc("/trigger-update", triggerUpdateHandler)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", nil))
}
