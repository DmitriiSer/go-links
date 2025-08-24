# Go Links - A Simple URL Shortener

A lightweight, self-hosted URL shortener inspired by the "Go Links" systems used internally at many companies. This service allows you to create simple, memorable aliases (like `go/pr`) that redirect to longer, more complex URLs.

It's built with Go and uses a local SQLite database, making it fast, portable, and easy to deploy with zero external dependencies.

## Features

- **Simple Redirects**: Turns `http://localhost:3000/<alias>` into a redirect to your destination URL.
- **Zero Dependencies**: Runs as a single binary with an embedded SQLite database. No CGo required, ensuring easy cross-compilation (e.g., for a Raspberry Pi).
- **Content Negotiation**:
  - Acts as a standard redirector for browsers.
  - Returns a JSON response for API clients that send an `Accept: application/json` header.
- **Easy to Deploy**: Just build and run the binary.

## Getting Started

### Prerequisites

- Go (version 1.18 or newer recommended)

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
    go run main.go
    ```

The server will start on `http://localhost:3000` and create a `links.db` file in the project directory to store the links.

## Usage

### Creating Links

Currently, links are added directly in the `main()` function in `main.go`. This is a temporary measure until the management UI is implemented.

To add a new link, use the `insertGoLink` function:

```go
// filepath: main.go
// ...existing code...
func main() {
// ...existing code...
    // For demonstration, let's add a sample link.
    insertGoLink("g", "https://google.com")
    insertGoLink("github", "https://github.com")
    insertGoLink("my-pr", "https://github.com/user/repo/pull/123") // Add your new link here

    mux := http.NewServeMux()
// ...existing code...
```

### Accessing Links

- **Browser Redirect**: Navigate to `http://localhost:3000/<alias>` in your browser. For example, `http://localhost:3000/g` will redirect to `https://google.com`.

- **API Request**: Use a tool like `curl` to get a JSON response.
  ```bash
  curl -H "Accept: application/json" http://localhost:3000/github
  ```
  **Response:**
  ```json
  {
    "path": "github",
    "url": "https://github.com"
  }
  ```

## Deployment Guide

For a real-world deployment example, see the detailed guide on setting up **Go Links** in a home network using _pfSense_ for DNS and a _Raspberry Pi_ with _Nginx_ as a reverse proxy.

➡️ **[Full Guide: Go Links with pfSense and Raspberry Pi](./docs/pfsense-raspberrypi-guide.md)**

## Project Status & Roadmap

This project is under active development. Here is a summary of completed features and planned enhancements.

### Completed Features

- [x] **Core Redirector Service**: The server correctly redirects aliases to their destination URLs.
- [x] **CGo-Free Database**: Uses a pure Go SQLite driver, ensuring easy cross-compilation.
- [x] **Deployment Guide**: Includes a detailed guide for a real-world deployment scenario.

### Planned Enhancements

The vision is to create a full-featured link management portal using a modern Go-based stack.

- [ ] **JSON API**: The server provides a JSON response for API clients via content negotiation.
- [ ] **Link Management Portal (`/go`)**
  - [ ] **Go + HTMX Stack**: Build the portal using Go's `html/template` package for server-side rendering, enhanced with _HTMX_ for dynamic UI interactions without page reloads.
  - [ ] **Modern Styling**: Integrate _Tailwind CSS_ to replicate the look and feel of modern UI components (like `shadcn/ui`).
  - [ ] **Full CRUD UI**: Implement the interface for creating, reading, updating, and deleting links.
- [ ] **Enhanced Validation**: Add robust server-side validation to prevent creating duplicate or invalid aliases.
- [ ] **Configuration**: Allow the port and database file path to be configured via environment variables or command-line flags.

## License

Distributed under the MIT License. See `LICENSE` for more information.
