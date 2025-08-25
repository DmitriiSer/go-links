package main

import (
	"log"
	"net/http"
	"strconv"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/go-openapi/spec"
)

func setupAPI(server *Server) *restful.Container {
	container := restful.NewContainer()

	ws := new(restful.WebService)
	ws.Path("/api").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

	// GET /api/links
	ws.Route(ws.GET("/links").
		To(func(req *restful.Request, resp *restful.Response) {
			server.apiLinksHandler(resp.ResponseWriter, req.Request)
		}).
		Doc("List links").
		Writes([]Link{}).
		Metadata(restfulspec.KeyOpenAPITags, []string{"links"}))

	// POST /api/links
	ws.Route(ws.POST("/links").
		To(func(req *restful.Request, resp *restful.Response) {
			server.apiLinksHandler(resp.ResponseWriter, req.Request)
		}).
		Doc("Create link").
		Reads(Link{}).
		Metadata(restfulspec.KeyOpenAPITags, []string{"links"}))

	// PUT /api/links/{id}
	ws.Route(ws.PUT("/links/{id}").
		To(func(req *restful.Request, resp *restful.Response) {
			idStr := req.PathParameter("id")
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				resp.WriteErrorString(http.StatusBadRequest, "invalid id")
				return
			}
			server.apiLinkIDHandler(resp.ResponseWriter, req.Request, id)
		}).
		Doc("Update link").
		Param(ws.PathParameter("id", "Link ID").DataType("integer")).
		Reads(Link{}).
		Metadata(restfulspec.KeyOpenAPITags, []string{"links"}))

	// DELETE /api/links/{id}
	ws.Route(ws.DELETE("/links/{id}").
		To(func(req *restful.Request, resp *restful.Response) {
			idStr := req.PathParameter("id")
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				resp.WriteErrorString(http.StatusBadRequest, "invalid id")
				return
			}
			server.apiLinkIDHandler(resp.ResponseWriter, req.Request, id)
		}).
		Doc("Delete link").
		Param(ws.PathParameter("id", "Link ID").DataType("integer")).
		Metadata(restfulspec.KeyOpenAPITags, []string{"links"}))

	container.Add(ws)

	// OpenAPI service mounted at /api/swagger/openapi.json
	cfg := restfulspec.Config{
		WebServices: []*restful.WebService{ws},
		APIPath:     "/api/swagger/openapi.json",
		PostBuildSwaggerObjectHandler: func(sw *spec.Swagger) {
			sw.Info = &spec.Info{InfoProps: spec.InfoProps{
				Title:       "Go Links API",
				Version:     "1.0",
				Description: "API for managing go links (CRUD and redirects)",
			}}
			// Keep BasePath empty because paths already include /api from ws.Path("/api")
			sw.BasePath = ""
			// Clear Host so UI uses current origin (prevents http://go/...)
			sw.Host = ""
			sw.Schemes = []string{"https"}
		},
	}
	container.Add(restfulspec.NewOpenAPIService(cfg))
	return container
}

func swaggerUIHandler(w http.ResponseWriter, r *http.Request) {
	// Minimal Swagger UI HTML pointing to our JSON endpoint
	html := `<!doctype html><html><head><meta charset="utf-8"/><title>Swagger UI</title>
	<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
	</head><body><div id="swagger"></div>
	<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
	<script>window.ui = SwaggerUIBundle({ url: '/api/swagger/openapi.json', dom_id: '#swagger' });</script>
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

	// Routes: /api via go-restful (auto OpenAPI), others via net/http
	apiContainer := setupAPI(server)
	mux := http.NewServeMux()
	mux.Handle("/api/", apiContainer)
	mux.HandleFunc("/swagger", swaggerUIHandler)
	mux.HandleFunc("/", server.rootHandler)

	port := "3000"
	log.Println("Server starting on port " + port + "...")
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		// Use log.Fatalf for consistency.
		log.Fatalf("Server failed to start: %v", err)
	}
}
