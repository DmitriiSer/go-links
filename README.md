# Go Links - A Simple URL Shortener

A lightweight, self-hosted URL shortener inspired by internal "Go Links" systems. Create simple, memorable aliases (like `go/pr`) that redirect to longer URLs. Built in Go with a local SQLite database—fast, portable, and easy to deploy.

## Features

- **Simple redirects**: Visit `http://localhost:3000/<alias>` to get redirected to the destination URL.
- **Runtime OpenAPI + Swagger UI**: API spec is generated at runtime; explore and test via Swagger UI.
- **REST JSON API**: Full CRUD for links under `/api`.
- **Pure Go SQLite**: Uses a CGo-free SQLite driver; easy cross-compilation and ARM-friendly.
- **Easy deploy**: Single binary; works well behind Nginx/HTTPS.

## Getting Started

### Prerequisites

- Go (1.20+ recommended)

### Installation & Running

1.  **Clone the repository:**

    ```bash
    git clone https://github.com/DmitriiSer/go-links
    cd go-links
    ```

2.  **Install dependencies:**

    ```bash
    go mod tidy
    ```

3.  **Run the server:**
    ```bash
    go run .
    ```

    The server starts on `http://localhost:3000` and creates `links.db` in the project directory.

    **Configuration options:**
    ```bash
    # Using environment variables
    PORT=8080 DB_PATH=/data/links.db go run .
    
    # Using command line flags
    go run . --port 8080 --db-path /data/links.db --host 127.0.0.1
    
    # Using short flags
    go run . -p 8080 -d /data/links.db -h 0.0.0.0
    
    # Show help
    go run . --help
    ```

    Optional (dev auto-reload):
    ```bash
    # Install wgo (file watcher for Go development)
    go install github.com/bokwoon95/wgo@latest
    
    # wgo wraps a command and reruns on file changes
    wgo go run .
    # If PATH issues occur: /home/<you>/go/bin/wgo go run .
    ```

## Configuration

Go Links supports flexible configuration via environment variables and command line flags. **Command line flags take precedence over environment variables.**

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `3000` |
| `HOST` | Server host (empty = all interfaces) | `` |
| `DB_PATH` | Database file path | `./links.db` |

### Command Line Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--port` | `-p` | Server port |
| `--host` | `-h` | Server host |
| `--db-path` | `-d` | Database file path |
| `--help` | | Show help information |

### Examples

```bash
# Production deployment
./go-links --port 80 --db-path /var/lib/go-links/links.db

# Development with different port
PORT=8080 go run .

# Docker/container deployment
docker run -e PORT=3000 -e DB_PATH=/data/links.db go-links

# Custom host binding
./go-links --host 127.0.0.1 --port 8080
```

## API and Docs

- **Swagger UI**: `http://localhost:3000/swagger` (or your configured port)
- **OpenAPI JSON**: `http://localhost:3000/api/swagger/openapi.json`

Notes for reverse proxy/HTTPS:
- The spec advertises `https` and leaves `host` empty so the UI uses the current origin. Works cleanly behind Nginx TLS termination.

### Endpoints (under `/api`)

- `GET /api/links` → List links
  ```bash
  curl http://localhost:3000/api/links
  ```

- `POST /api/links` → Create link
  ```bash
  curl -X POST http://localhost:3000/api/links \
    -H 'Content-Type: application/json' \
    -d '{"path":"g","url":"https://google.com"}'
  ```
  - Validation: rejects empty/malformed URLs, non-http(s) schemes, and missing host (400).

- `PUT /api/links/{id}` → Update link
  ```bash
  curl -X PUT http://localhost:3000/api/links/1 \
    -H 'Content-Type: application/json' \
    -d '{"path":"g","url":"https://google.com"}'
  ```

- `DELETE /api/links/{id}` → Delete link
  ```bash
  curl -X DELETE http://localhost:3000/api/links/1
  ```

### Redirects

Navigate to `http://localhost:3000/<alias>` (e.g., `http://localhost:3000/g`) to be redirected to the configured URL.

## Deployment Guide

For a real-world deployment example, see the detailed guide on setting up **Go Links** in a home network using _pfSense_ for DNS and a _Raspberry Pi_ with _Nginx_ as a reverse proxy.

➡️ **[Full Guide: Go Links with pfSense and Raspberry Pi](./docs/pfsense-raspberrypi-guide.md)**

## Project Status & Roadmap

This project is under active development. Here is a summary of completed features and planned enhancements.

### Completed

- [x] Core redirector service
- [x] Pure Go SQLite store
- [x] Runtime OpenAPI generation and Swagger UI
- [x] CRUD JSON API under `/api`
- [x] Comprehensive input validation (URL schemes, path rules, reserved words)
- [x] Advanced error handling (proper HTTP status codes, structured JSON responses)
- [x] Uniqueness validation with clear feedback
- [x] Flexible configuration (environment variables, command line flags)
- [x] Deployment guide (pfSense + Raspberry Pi + Nginx)

### Planned

**Link Management Portal at `/go` (SSR templates + HTMX)**
- [x] Phase 1: Template Foundation (base layout, Tailwind CSS, data display)
- [ ] Phase 2: Static Portal (render real data, basic forms)  
- [ ] Phase 3: HTMX Integration (dynamic interactions, real-time search)
- [ ] Phase 4: Polish & Enhancement (UX improvements, accessibility)

📋 **[Full Implementation Plan](./docs/go-portal-implementation-plan.md)**

## License

Distributed under the MIT License. See `LICENSE` for more information.
