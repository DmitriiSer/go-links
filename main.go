package main

import (
	"log"
	"net/http"
)

func main() {
	// Initialize the database store.
	store, err := NewStore("./links.db")
	if err != nil {
		log.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Initialize the server with the store.
	server, err := NewServer(store)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Set up the HTTP server.
	mux := http.NewServeMux()
	mux.HandleFunc("/", server.rootHandler)

	port := "3000"
	log.Println("Server starting on port " + port + "...")
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		// Use log.Fatalf for consistency.
		log.Fatalf("Server failed to start: %v", err)
	}
}
