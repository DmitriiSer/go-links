package main

import (
	"database/sql"
	"log"
	"net/http"

	_ "modernc.org/sqlite"
)

var db *sql.DB

func getGoLinkFromAlias(alias string) (string, error) {
	var url string
	err := db.QueryRow("SELECT url FROM links WHERE path = ?", alias).Scan(&url)
	if err != nil {
		return "", err
	}
	return url, nil
}

func insertGoLink(alias string, url string) error {
	// insert or ignore to avoid duplicate entries
	_, err := db.Exec("INSERT OR IGNORE INTO links (path, url) VALUES (?, ?)", alias, url)
	return err
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	alias := r.URL.Path[1:]

	if r.Method == http.MethodGet {
		link, err := getGoLinkFromAlias(alias)
		if err != nil {
			log.Printf("No link found for alias: %s\n", alias)
			http.Error(w, "No link found", http.StatusNotFound)
			return
		}
		log.Printf("Found the following link for alias '%s': %s\n", alias, link)
		http.Redirect(w, r, link, http.StatusFound)
		return
	} else {
		return
	}
}

func main() {
	var err error
	// Open the SQLite database. It will be created if it doesn't exist.
	db, err = sql.Open("sqlite", "./links.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create the links table if it doesn't already exist.
	createTableSQL := `CREATE TABLE IF NOT EXISTS links (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"path" TEXT NOT NULL UNIQUE,
		"url" TEXT NOT NULL
	);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	// For demonstration, let's add a sample link.
	insertGoLink("g", "https://google.com")
	insertGoLink("github", "https://github.com")

	mux := http.NewServeMux()
	mux.HandleFunc("/", rootHandler)
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {})

	port := "3000"
	log.Println("Server starting on port " + port + "...")
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		panic(err)
	}
}
