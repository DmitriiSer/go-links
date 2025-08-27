package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// Config holds all configuration for the application.
type Config struct {
	Port   string
	Host   string
	DBPath string
}

// LoadConfig loads configuration from environment variables and command line flags.
// Priority: command line flags > environment variables > defaults.
func LoadConfig() (*Config, error) {
	config := &Config{
		Port:   "3000",           // Default port
		Host:   "",               // Default to all interfaces
		DBPath: "./links.db",     // Default database path
	}

	// Load from environment variables first
	if port := os.Getenv("PORT"); port != "" {
		config.Port = port
	}
	if host := os.Getenv("HOST"); host != "" {
		config.Host = host
	}
	if dbPath := os.Getenv("DB_PATH"); dbPath != "" {
		config.DBPath = dbPath
	}

	// Define command line flags (these override environment variables)
	var (
		portFlag   = flag.String("port", config.Port, "Server port (can also be set via PORT env var)")
		pFlag      = flag.String("p", "", "Server port (shorthand)")
		hostFlag   = flag.String("host", config.Host, "Server host (can also be set via HOST env var)")
		hFlag      = flag.String("h", "", "Server host (shorthand)")
		dbPathFlag = flag.String("db-path", config.DBPath, "Database file path (can also be set via DB_PATH env var)")
		dFlag      = flag.String("d", "", "Database file path (shorthand)")
		helpFlag   = flag.Bool("help", false, "Show help information")
	)

	// Custom usage function
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Go Links - A Simple URL Shortener\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  PORT      Server port (default: 3000)\n")
		fmt.Fprintf(os.Stderr, "  HOST      Server host (default: all interfaces)\n")
		fmt.Fprintf(os.Stderr, "  DB_PATH   Database file path (default: ./links.db)\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s --port 8080 --db-path /data/links.db\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  PORT=8080 %s\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -p 8080 -d /tmp/links.db\n", os.Args[0])
	}

	flag.Parse()

	// Show help if requested
	if *helpFlag {
		flag.Usage()
		os.Exit(0)
	}

	// Apply command line flags (override environment variables)
	if *portFlag != config.Port {
		config.Port = *portFlag
	}
	if *pFlag != "" {
		config.Port = *pFlag
	}
	if *hostFlag != config.Host {
		config.Host = *hostFlag
	}
	if *hFlag != "" {
		config.Host = *hFlag
	}
	if *dbPathFlag != config.DBPath {
		config.DBPath = *dbPathFlag
	}
	if *dFlag != "" {
		config.DBPath = *dFlag
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate checks if the configuration values are valid.
func (c *Config) Validate() error {
	// Validate port
	if port, err := strconv.Atoi(c.Port); err != nil {
		return fmt.Errorf("invalid port '%s': must be a number", c.Port)
	} else if port < 1 || port > 65535 {
		return fmt.Errorf("invalid port %d: must be between 1 and 65535", port)
	}

	// Validate database path
	if c.DBPath == "" {
		return fmt.Errorf("database path cannot be empty")
	}

	// Check if database directory exists (create if possible)
	dbDir := filepath.Dir(c.DBPath)
	if dbDir != "." && dbDir != "/" {
		if _, err := os.Stat(dbDir); os.IsNotExist(err) {
			if err := os.MkdirAll(dbDir, 0755); err != nil {
				return fmt.Errorf("cannot create database directory '%s': %v", dbDir, err)
			}
		}
	}

	return nil
}

// Address returns the full address string for the HTTP server.
func (c *Config) Address() string {
	return c.Host + ":" + c.Port
}

// String returns a string representation of the configuration.
func (c *Config) String() string {
	return fmt.Sprintf("Config{Port: %s, Host: %s, DBPath: %s}", c.Port, c.Host, c.DBPath)
}
