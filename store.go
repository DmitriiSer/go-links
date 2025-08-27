package main

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

// Store manages the database operations for links.
type Store struct {
	db *sql.DB
}

// Link represents a shortened URL link.
type Link struct {
	ID   int64  `json:"id"`
	Path string `json:"path"`
	URL  string `json:"url"`
}

// NewStore creates a new Store and initializes the database.
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create the links table if it doesn't already exist.
	createTableSQL := `CREATE TABLE IF NOT EXISTS links (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"path" TEXT NOT NULL UNIQUE,
		"url" TEXT NOT NULL
	);`
	if _, err := db.Exec(createTableSQL); err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the database connection.
func (s *Store) Close() {
	s.db.Close()
}

// GetLinkByPath retrieves a single link by its path.
func (s *Store) GetLinkByPath(path string) (*Link, error) {
	link := &Link{}
	err := s.db.QueryRow("SELECT id, path, url FROM links WHERE path = ?", path).Scan(&link.ID, &link.Path, &link.URL)
	if err != nil {
		return nil, err
	}
	return link, nil
}

// GetAllLinks retrieves all links from the database.
func (s *Store) GetAllLinks() ([]Link, error) {
	rows, err := s.db.Query("SELECT id, path, url FROM links ORDER BY path")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []Link
	for rows.Next() {
		var link Link
		if err := rows.Scan(&link.ID, &link.Path, &link.URL); err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, nil
}

// CreateLink adds a new link to the database.
func (s *Store) CreateLink(path, url string) error {
	insertSQL := `INSERT INTO links(path, url) VALUES(?, ?)`
	_, err := s.db.Exec(insertSQL, path, url)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: links.path") {
			return fmt.Errorf("a link with path '%s' already exists", path)
		}
		return err
	}
	return nil
}

// UpdateLink updates an existing link.
func (s *Store) UpdateLink(id int64, path, url string) error {
	updateSQL := `UPDATE links SET path = ?, url = ? WHERE id = ?`
	_, err := s.db.Exec(updateSQL, path, url, id)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: links.path") {
			return fmt.Errorf("a link with path '%s' already exists", path)
		}
		return err
	}
	return nil
}

// LinkExists checks if a link with the given ID exists.
func (s *Store) LinkExists(id int64) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM links WHERE id = ?)`
	err := s.db.QueryRow(query, id).Scan(&exists)
	return exists, err
}

// DeleteLink removes a link from the database by its ID.
func (s *Store) DeleteLink(id int64) error {
	deleteSQL := `DELETE FROM links WHERE id = ?`
	result, err := s.db.Exec(deleteSQL, id)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("link with id %d not found", id)
	}
	
	return nil
}
