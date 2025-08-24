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

## Roadmap

The vision is to create a full-featured link management portal.

- [ ] **Web UI**: Create a web interface at a dedicated path (e.g., `/manage`) for CRUD (Create, Read, Update, Delete) operations on links.
- [ ] **Link Validation**: Ensure that users can only create aliases for paths that are not already taken.
- [ ] **Configuration**: Allow the port and database file path to be configured via environment variables or command-line flags.

## License

Distributed under the MIT License. See `LICENSE` for more information.
