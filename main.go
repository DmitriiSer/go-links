package main

import (
	"log"
	"net/http"
	"encoding/json"
	"reflect"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
)

var openAPIDoc *openapi3.T

func initOpenAPIDoc() {
	// Build OpenAPI 3 document at runtime
	doc := &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:       "Go Links API",
			Version:     "1.0",
			Description: "API for managing go links (CRUD and redirects)",
		},
		Servers: openapi3.Servers{{URL: "/api"}},
	}

	// Initialize components and paths (pointers required)
	components := openapi3.NewComponents()
	doc.Components = &components
	doc.Paths = openapi3.NewPaths()
	// Initialize schema map to avoid nil map assignment
	if doc.Components.Schemas == nil {
		doc.Components.Schemas = make(openapi3.Schemas)
	}

	// Schemas from Go types
	gen := openapi3gen.NewGenerator()
	if linkSchemaRef, err := gen.GenerateSchemaRef(reflect.TypeOf(Link{})); err == nil {
		doc.Components.Schemas["Link"] = linkSchemaRef
	}

	// GET /links
	getLinks := openapi3.NewOperation()
	getLinks.Summary = "List links"
	getLinks.Description = "Retrieve all stored links"
	getLinks.Tags = []string{"links"}
	getLinks.AddResponse(200, openapi3.NewResponse().
		WithDescription("OK").
		WithJSONSchema(&openapi3.Schema{Type: "array", Items: &openapi3.SchemaRef{Ref: "#/components/schemas/Link"}}))
	// POST /links
	createLink := openapi3.NewOperation()
	createLink.Summary = "Create a link"
	createLink.Description = "Create a new link"
	createLink.Tags = []string{"links"}
	createLink.RequestBody = &openapi3.RequestBodyRef{Value: &openapi3.RequestBody{
		Required: true,
		Content:  openapi3.NewContentWithJSONSchemaRef(&openapi3.SchemaRef{Ref: "#/components/schemas/Link"}),
	}}
	createLink.AddResponse(201, openapi3.NewResponse().WithDescription("Created"))
	createLink.AddResponse(400, openapi3.NewResponse().WithDescription("Invalid request body"))
	createLink.AddResponse(500, openapi3.NewResponse().WithDescription("Failed to create link"))

	doc.AddOperation("/links", http.MethodGet, getLinks)
	doc.AddOperation("/links", http.MethodPost, createLink)

	// PUT /links/{id}
	updateLink := openapi3.NewOperation()
	updateLink.Summary = "Update a link"
	updateLink.Description = "Update an existing link by ID"
	updateLink.Tags = []string{"links"}
	updateLink.AddParameter(openapi3.NewPathParameter("id").WithSchema(&openapi3.Schema{Type: "integer"}))
	updateLink.RequestBody = &openapi3.RequestBodyRef{Value: &openapi3.RequestBody{
		Required: true,
		Content:  openapi3.NewContentWithJSONSchemaRef(&openapi3.SchemaRef{Ref: "#/components/schemas/Link"}),
	}}
	updateLink.AddResponse(200, openapi3.NewResponse().WithDescription("OK"))
	updateLink.AddResponse(400, openapi3.NewResponse().WithDescription("Invalid request body"))
	updateLink.AddResponse(500, openapi3.NewResponse().WithDescription("Failed to update link"))

	// DELETE /links/{id}
	deleteLink := openapi3.NewOperation()
	deleteLink.Summary = "Delete a link"
	deleteLink.Description = "Delete a link by ID"
	deleteLink.Tags = []string{"links"}
	deleteLink.AddParameter(openapi3.NewPathParameter("id").WithSchema(&openapi3.Schema{Type: "integer"}))
	deleteLink.AddResponse(200, openapi3.NewResponse().WithDescription("OK"))
	deleteLink.AddResponse(500, openapi3.NewResponse().WithDescription("Failed to delete link"))

	doc.AddOperation("/links/{id}", http.MethodPut, updateLink)
	doc.AddOperation("/links/{id}", http.MethodDelete, deleteLink)

	openAPIDoc = doc
}

// func strPtr(s string) *string { return &s }

func openAPIJSONHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(openAPIDoc)
}

func swaggerUIHandler(w http.ResponseWriter, r *http.Request) {
	// Minimal Swagger UI HTML pointing to our JSON endpoint
	html := `<!doctype html><html><head><meta charset="utf-8"/><title>Swagger UI</title>
	<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
	</head><body><div id="swagger"></div>
	<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
	<script>window.ui = SwaggerUIBundle({ url: '/swagger/openapi.json', dom_id: '#swagger' });</script>
	</body></html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}

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

	// Build OpenAPI doc and set up the HTTP server.
	initOpenAPIDoc()
	mux := http.NewServeMux()
	mux.HandleFunc("/swagger/openapi.json", openAPIJSONHandler)
	mux.HandleFunc("/swagger", swaggerUIHandler)
	mux.HandleFunc("/", server.rootHandler)

	port := "3000"
	log.Println("Server starting on port " + port + "...")
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		// Use log.Fatalf for consistency.
		log.Fatalf("Server failed to start: %v", err)
	}
}
