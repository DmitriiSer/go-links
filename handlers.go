package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// Server holds the dependencies for the web application.
type Server struct {
	store     *Store
	templates *template.Template
}

// NewServer creates a new Server with necessary dependencies.
func NewServer(store *Store) (*Server, error) {
	// Parse template files from the templates directory
	templates, err := template.ParseGlob("templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("error parsing templates: %w", err)
	}

	// Parse component templates
	componentTemplates, err := template.ParseGlob("templates/components/*.html")
	if err != nil {
		// Components are optional for now, just log the error
		log.Printf("Warning: Could not parse component templates: %v", err)
	} else {
		// Add component templates to the main template
		for _, t := range componentTemplates.Templates() {
			templates, err = templates.AddParseTree(t.Name(), t.Tree)
			if err != nil {
				log.Printf("Warning: Could not add component template %s: %v", t.Name(), err)
			}
		}
	}

	return &Server{
		store:     store,
		templates: templates,
	}, nil
}

// rootHandler is the main entry point for all requests.
func (s *Server) rootHandler(w http.ResponseWriter, r *http.Request) {
	// Handle portal requests
	if r.URL.Path == "/go" {
		s.goPortalHandler(w, r)
		return
	}
	
	// Handle portal link management (traditional forms)
	if strings.HasPrefix(r.URL.Path, "/go/links") {
		s.goLinksRouter(w, r)
		return
	}
	
	// Handle HTMX portal requests
	if strings.HasPrefix(r.URL.Path, "/go/htmx") {
		s.htmxRouter(w, r)
		return
	}

	// Handle API requests
	if strings.HasPrefix(r.URL.Path, "/api/") {
		s.apiRouter(w, r)
		return
	}

	// Handle favicon requests
	if r.URL.Path == "/favicon.ico" {
		http.NotFound(w, r)
		return
	}

	// Handle redirects
	s.redirectHandler(w, r)
}

// goLinksRouter handles /go/links/* routes for CRUD operations
func (s *Server) goLinksRouter(w http.ResponseWriter, r *http.Request) {
	// Parse the path to extract ID if present
	path := strings.TrimPrefix(r.URL.Path, "/go/links")
	if path == "" {
		// /go/links - create new link
		if r.Method == http.MethodPost {
			s.handlePortalPost(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	
	// /go/links/{id} - edit/delete link
	if path[0] == '/' {
		idStr := path[1:]
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid link ID", http.StatusBadRequest)
			return
		}
		
		// Handle method override for PUT/DELETE via forms
		method := r.Method
		if r.Method == http.MethodPost {
			if methodOverride := r.FormValue("_method"); methodOverride != "" {
				method = methodOverride
			}
		}
		
		switch method {
		case http.MethodPut:
			s.handlePortalUpdate(w, r, id)
		case http.MethodDelete:
			s.handlePortalDelete(w, r, id)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	
	http.NotFound(w, r)
}

// handlePortalUpdate handles updating a link via the portal
func (s *Server) handlePortalUpdate(w http.ResponseWriter, r *http.Request, id int64) {
	// Parse form data
	err := r.ParseForm()
	if err != nil {
		log.Printf("Error parsing form: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Get form values
	path := strings.TrimSpace(r.FormValue("path"))
	url := strings.TrimSpace(r.FormValue("url"))

	// Create link object for validation
	link := Link{
		ID:   id,
		Path: path,
		URL:  url,
	}

	// Validate the link
	errors := make(map[string]string)
	if err := validateLink(link); err != nil {
		errors["General"] = err.Error()
	}

	// If validation passes, update the link
	if len(errors) == 0 {
		err = s.store.UpdateLink(id, path, url)
		if err != nil {
			log.Printf("Error updating link: %v", err)
			if strings.Contains(err.Error(), "already exists") {
				errors["Path"] = err.Error()
			} else {
				errors["General"] = "Failed to update link"
			}
		} else {
			// Success - redirect
			http.Redirect(w, r, "/go?success=Link updated successfully", http.StatusSeeOther)
			return
		}
	}

	// If we get here, there were errors
	s.renderPortalWithForm(w, r, link, errors, true, true, "")
}

// handlePortalDelete handles deleting a link via the portal
func (s *Server) handlePortalDelete(w http.ResponseWriter, r *http.Request, id int64) {
	err := s.store.DeleteLink(id)
	if err != nil {
		log.Printf("Error deleting link: %v", err)
		if strings.Contains(err.Error(), "not found") {
			http.Redirect(w, r, "/go?error=Link not found", http.StatusSeeOther)
		} else {
			http.Redirect(w, r, "/go?error=Failed to delete link", http.StatusSeeOther)
		}
		return
	}

	// Success
	http.Redirect(w, r, "/go?success=Link deleted successfully", http.StatusSeeOther)
}

// htmxRouter handles /go/htmx/* routes for HTMX dynamic requests
func (s *Server) htmxRouter(w http.ResponseWriter, r *http.Request) {
	// Parse the path
	path := strings.TrimPrefix(r.URL.Path, "/go/htmx")
	
	if path == "/search" {
		s.htmxSearchHandler(w, r)
		return
	}
	
	if strings.HasPrefix(path, "/links") {
		s.htmxLinksRouter(w, r, path)
		return
	}
	
	http.NotFound(w, r)
}

// htmxLinksRouter handles /go/htmx/links/* routes
func (s *Server) htmxLinksRouter(w http.ResponseWriter, r *http.Request, path string) {
	linksPath := strings.TrimPrefix(path, "/links")
	
	if linksPath == "" {
		// /go/htmx/links - create new link
		if r.Method == http.MethodPost {
			s.htmxCreateLink(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	
	if linksPath == "/new" {
		// /go/htmx/links/new - show new link form
		if r.Method == http.MethodGet {
			s.htmxNewLinkForm(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	
	// /go/htmx/links/{id} or /go/htmx/links/{id}/edit
	if linksPath[0] == '/' {
		parts := strings.Split(linksPath[1:], "/")
		if len(parts) == 0 {
			http.NotFound(w, r)
			return
		}
		
		id, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			http.Error(w, "Invalid link ID", http.StatusBadRequest)
			return
		}
		
		if len(parts) == 2 && parts[1] == "edit" {
			// /go/htmx/links/{id}/edit - show edit form
			if r.Method == http.MethodGet {
				s.htmxEditLinkForm(w, r, id)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}
		
		if len(parts) == 1 {
			// /go/htmx/links/{id} - update or delete
			switch r.Method {
			case http.MethodPut:
				s.htmxUpdateLink(w, r, id)
			case http.MethodDelete:
				s.htmxDeleteLink(w, r, id)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}
	}
	
	http.NotFound(w, r)
}

// htmxSearchHandler handles real-time search requests
func (s *Server) htmxSearchHandler(w http.ResponseWriter, r *http.Request) {
	searchQuery := r.URL.Query().Get("search")
	
	// Get all links from the database
	links, err := s.store.GetAllLinks()
	if err != nil {
		log.Printf("Error fetching links for search: %v", err)
		http.Error(w, "Failed to search links", http.StatusInternalServerError)
		return
	}

	// Filter links if search query provided
	if searchQuery != "" {
		filteredLinks := []Link{}
		for _, link := range links {
			if strings.Contains(strings.ToLower(link.Path), strings.ToLower(searchQuery)) ||
			   strings.Contains(strings.ToLower(link.URL), strings.ToLower(searchQuery)) {
				filteredLinks = append(filteredLinks, link)
			}
		}
		links = filteredLinks
	}

	// Prepare data for the link-list template
	data := struct {
		Links []Link
	}{
		Links: links,
	}

	// Render only the link-list component
	err = s.templates.ExecuteTemplate(w, "link-list", data)
	if err != nil {
		log.Printf("Template execution error in search: %v", err)
		http.Error(w, "Template rendering error", http.StatusInternalServerError)
		return
	}
}

// htmxNewLinkForm shows the new link form
func (s *Server) htmxNewLinkForm(w http.ResponseWriter, r *http.Request) {
	data := struct {
		ShowForm bool
		EditMode bool
		Link     Link
		Errors   map[string]string
	}{
		ShowForm: true,
		EditMode: false,
		Link:     Link{},
		Errors:   make(map[string]string),
	}

	err := s.templates.ExecuteTemplate(w, "link-form", data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Template rendering error", http.StatusInternalServerError)
		return
	}
}

// htmxEditLinkForm shows the edit link form
func (s *Server) htmxEditLinkForm(w http.ResponseWriter, r *http.Request, id int64) {
	// Get the link from database
	links, err := s.store.GetAllLinks()
	if err != nil {
		log.Printf("Error fetching links: %v", err)
		http.Error(w, "Failed to load link", http.StatusInternalServerError)
		return
	}

	var link Link
	found := false
	for _, l := range links {
		if l.ID == id {
			link = l
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "Link not found", http.StatusNotFound)
		return
	}

	data := struct {
		ShowForm bool
		EditMode bool
		Link     Link
		Errors   map[string]string
	}{
		ShowForm: true,
		EditMode: true,
		Link:     link,
		Errors:   make(map[string]string),
	}

	err = s.templates.ExecuteTemplate(w, "link-form", data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Template rendering error", http.StatusInternalServerError)
		return
	}
}

// htmxCreateLink handles creating a link via HTMX
func (s *Server) htmxCreateLink(w http.ResponseWriter, r *http.Request) {
	// Parse form data
	err := r.ParseForm()
	if err != nil {
		log.Printf("Error parsing form: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Get form values
	path := strings.TrimSpace(r.FormValue("path"))
	url := strings.TrimSpace(r.FormValue("url"))

	// Create link object for validation
	link := Link{
		Path: path,
		URL:  url,
	}

	// Validate the link
	errors := make(map[string]string)
	if err := validateLink(link); err != nil {
		errors["General"] = err.Error()
	}

	// If validation passes, create the link
	if len(errors) == 0 {
		err = s.store.CreateLink(path, url)
		if err != nil {
			log.Printf("Error creating link: %v", err)
			if strings.Contains(err.Error(), "already exists") {
				errors["Path"] = err.Error()
			} else {
				errors["General"] = "Failed to create link"
			}
		} else {
			// Success - return the updated portal content
			s.htmxRenderPortalContent(w, r, "Link created successfully", "")
			return
		}
	}

	// If we get here, there were errors - redisplay form with errors
	data := struct {
		ShowForm bool
		EditMode bool
		Link     Link
		Errors   map[string]string
	}{
		ShowForm: true,
		EditMode: false,
		Link:     link,
		Errors:   errors,
	}

	err = s.templates.ExecuteTemplate(w, "link-form", data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Template rendering error", http.StatusInternalServerError)
		return
	}
}

// htmxUpdateLink handles updating a link via HTMX
func (s *Server) htmxUpdateLink(w http.ResponseWriter, r *http.Request, id int64) {
	// Parse form data
	err := r.ParseForm()
	if err != nil {
		log.Printf("Error parsing form: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Get form values
	path := strings.TrimSpace(r.FormValue("path"))
	url := strings.TrimSpace(r.FormValue("url"))

	// Create link object for validation
	link := Link{
		ID:   id,
		Path: path,
		URL:  url,
	}

	// Validate the link
	errors := make(map[string]string)
	if err := validateLink(link); err != nil {
		errors["General"] = err.Error()
	}

	// If validation passes, update the link
	if len(errors) == 0 {
		err = s.store.UpdateLink(id, path, url)
		if err != nil {
			log.Printf("Error updating link: %v", err)
			if strings.Contains(err.Error(), "already exists") {
				errors["Path"] = err.Error()
			} else {
				errors["General"] = "Failed to update link"
			}
		} else {
			// Success - return the updated portal content
			s.htmxRenderPortalContent(w, r, "Link updated successfully", "")
			return
		}
	}

	// If we get here, there were errors - redisplay form with errors
	data := struct {
		ShowForm bool
		EditMode bool
		Link     Link
		Errors   map[string]string
	}{
		ShowForm: true,
		EditMode: true,
		Link:     link,
		Errors:   errors,
	}

	err = s.templates.ExecuteTemplate(w, "link-form", data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Template rendering error", http.StatusInternalServerError)
		return
	}
}

// htmxDeleteLink handles deleting a link via HTMX
func (s *Server) htmxDeleteLink(w http.ResponseWriter, r *http.Request, id int64) {
	err := s.store.DeleteLink(id)
	if err != nil {
		log.Printf("Error deleting link: %v", err)
		if strings.Contains(err.Error(), "not found") {
			s.htmxRenderPortalContent(w, r, "", "Link not found")
		} else {
			s.htmxRenderPortalContent(w, r, "", "Failed to delete link")
		}
		return
	}

	// Success
	s.htmxRenderPortalContent(w, r, "Link deleted successfully", "")
}

// htmxRenderPortalContent renders the entire portal content with messages
func (s *Server) htmxRenderPortalContent(w http.ResponseWriter, r *http.Request, successMessage, errorMessage string) {
	// Get all links for display
	links, err := s.store.GetAllLinks()
	if err != nil {
		log.Printf("Error fetching links for portal: %v", err)
		http.Error(w, "Failed to load links", http.StatusInternalServerError)
		return
	}

	// Calculate dashboard stats
	var mostRecentLink string
	if len(links) > 0 {
		mostRecentLink = links[len(links)-1].Path
	}

	// Prepare template data
	data := PortalData{
		Title:           "Portal",
		PageHeader:      "Link Management Portal",
		PageDescription: "Manage your go links with ease",
		ShowDashboard:   true,
		Links:           links,
		LinkCount:       len(links),
		MostPopularLink: mostRecentLink,
		DatabaseStatus:  "OK",
		ShowForm:        false,
		EditMode:        false,
		Errors:          make(map[string]string),
		SuccessMessage:  successMessage,
		ErrorMessage:    errorMessage,
	}

	// Render the portal content template
	err = s.templates.ExecuteTemplate(w, "content", data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Template rendering error", http.StatusInternalServerError)
		return
	}
}

// redirectHandler handles the URL redirection logic.
func (s *Server) redirectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Strip the leading slash from the path to match database storage
	path := strings.TrimPrefix(r.URL.Path, "/")

	link, err := s.store.GetLinkByPath(path)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		log.Printf("Database error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, link.URL, http.StatusFound)
}

// PortalData holds data for the portal template.
type PortalData struct {
	Title           string
	PageHeader      string
	PageDescription string
	ShowDashboard   bool
	Links           []Link
	LinkCount       int
	MostPopularLink string
	DatabaseStatus  string
	SearchQuery     string
	ShowForm        bool
	EditMode        bool
	Link            Link
	Errors          map[string]string
	SuccessMessage  string
	ErrorMessage    string
	InfoMessage     string
}

// goPortalHandler serves the main management UI.
func (s *Server) goPortalHandler(w http.ResponseWriter, r *http.Request) {
	// Handle different HTTP methods
	switch r.Method {
	case http.MethodGet:
		s.handlePortalGet(w, r)
	case http.MethodPost:
		s.handlePortalPost(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handlePortalGet displays the portal page
func (s *Server) handlePortalGet(w http.ResponseWriter, r *http.Request) {
	// Get search query if any
	searchQuery := r.URL.Query().Get("search")
	
	// Get all links from the database
	links, err := s.store.GetAllLinks()
	if err != nil {
		log.Printf("Error fetching links for portal: %v", err)
		writeErrorJSON(w, "Failed to load links", http.StatusInternalServerError)
		return
	}

	// Filter links if search query provided
	if searchQuery != "" {
		filteredLinks := []Link{}
		for _, link := range links {
			if strings.Contains(strings.ToLower(link.Path), strings.ToLower(searchQuery)) ||
			   strings.Contains(strings.ToLower(link.URL), strings.ToLower(searchQuery)) {
				filteredLinks = append(filteredLinks, link)
			}
		}
		links = filteredLinks
	}

	// Calculate dashboard stats
	var mostRecentLink string
	if len(links) > 0 {
		// Use the last link in the sorted list as "most recent"
		// (links are ordered by path, so we'll use the last one for now)
		mostRecentLink = links[len(links)-1].Path
	}

	// Check for messages in URL
	successMessage := r.URL.Query().Get("success")
	errorMessage := r.URL.Query().Get("error")

	// Prepare template data
	data := PortalData{
		Title:           "Portal",
		PageHeader:      "Link Management Portal", 
		PageDescription: "Manage your go links with ease",
		ShowDashboard:   true,
		Links:           links,
		LinkCount:       len(links),
		MostPopularLink: mostRecentLink,
		DatabaseStatus:  "OK",
		SearchQuery:     searchQuery,
		ShowForm:        false,
		EditMode:        false,
		Errors:          make(map[string]string),
		SuccessMessage:  successMessage,
		ErrorMessage:    errorMessage,
	}

	// Render the portal template
	err = s.templates.ExecuteTemplate(w, "base.html", data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		writeErrorJSON(w, "Template rendering error", http.StatusInternalServerError)
		return
	}
}

// handlePortalPost handles form submissions for creating links
func (s *Server) handlePortalPost(w http.ResponseWriter, r *http.Request) {
	// Parse form data
	err := r.ParseForm()
	if err != nil {
		log.Printf("Error parsing form: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Get form values
	path := strings.TrimSpace(r.FormValue("path"))
	url := strings.TrimSpace(r.FormValue("url"))

	// Create link object for validation
	link := Link{
		Path: path,
		URL:  url,
	}

	// Validate the link
	errors := make(map[string]string)
	if err := validateLink(link); err != nil {
		// Parse validation error
		errors["General"] = err.Error()
	}

	// If validation passes, create the link
	if len(errors) == 0 {
		err = s.store.CreateLink(path, url)
		if err != nil {
			log.Printf("Error creating link: %v", err)
			if strings.Contains(err.Error(), "already exists") {
				errors["Path"] = err.Error()
			} else {
				errors["General"] = "Failed to create link"
			}
		} else {
			// Success - redirect to avoid resubmission
			http.Redirect(w, r, "/go?success=Link created successfully", http.StatusSeeOther)
			return
		}
	}

	// If we get here, there were errors - redisplay form with errors
	s.renderPortalWithForm(w, r, link, errors, true, false, "")
}

// renderPortalWithForm renders the portal with the form visible and any messages
func (s *Server) renderPortalWithForm(w http.ResponseWriter, r *http.Request, link Link, errors map[string]string, showForm bool, editMode bool, successMessage string) {
	// Get all links for display
	links, err := s.store.GetAllLinks()
	if err != nil {
		log.Printf("Error fetching links for portal: %v", err)
		writeErrorJSON(w, "Failed to load links", http.StatusInternalServerError)
		return
	}

	// Calculate dashboard stats
	var mostRecentLink string
	if len(links) > 0 {
		mostRecentLink = links[len(links)-1].Path
	}

	// Check for success message in URL
	if successMessage == "" {
		successMessage = r.URL.Query().Get("success")
	}
	
	// Check for error message in URL
	errorMessage := r.URL.Query().Get("error")

	// Prepare template data
	data := PortalData{
		Title:           "Portal",
		PageHeader:      "Link Management Portal",
		PageDescription: "Manage your go links with ease",
		ShowDashboard:   true,
		Links:           links,
		LinkCount:       len(links),
		MostPopularLink: mostRecentLink,
		DatabaseStatus:  "OK",
		ShowForm:        showForm,
		EditMode:        editMode,
		Link:            link,
		Errors:          errors,
		SuccessMessage:  successMessage,
		ErrorMessage:    errorMessage,
	}

	// Render the portal template
	err = s.templates.ExecuteTemplate(w, "base.html", data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		writeErrorJSON(w, "Template rendering error", http.StatusInternalServerError)
		return
	}
}

// apiRouter routes API requests to the correct handler.
func (s *Server) apiRouter(w http.ResponseWriter, r *http.Request) {
	// Expecting paths like /api/links or /api/links/123
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 || parts[1] != "links" {
		http.NotFound(w, r)
		return
	}

	// /api/links/{id}
	if len(parts) == 3 {
		id, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			writeErrorJSON(w, "Invalid link ID", http.StatusBadRequest)
			return
		}
		s.apiLinkIDHandler(w, r, id)
		return
	}

	// /api/links
	if len(parts) == 2 {
		s.apiLinksHandler(w, r)
		return
	}

	http.NotFound(w, r)
}

// apiLinksHandler handles requests for the /api/links collection.
func (s *Server) apiLinksHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetLinks(w, r)
	case http.MethodPost:
		s.handleCreateLink(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// apiLinkIDHandler handles requests for a specific link by its ID.
func (s *Server) apiLinkIDHandler(w http.ResponseWriter, r *http.Request, id int64) {
	switch r.Method {
	case http.MethodPut:
		s.handleUpdateLink(w, r, id)
	case http.MethodDelete:
		s.handleDeleteLink(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetLinks retrieves all links and returns them as JSON.
// GetLinks godoc
// @Summary      List links
// @Description  Retrieve all stored links
// @Tags         links
// @Produce      json
// @Success      200  {array}   Link
// @Router       /links [get]
func (s *Server) handleGetLinks(w http.ResponseWriter, r *http.Request) {
	links, err := s.store.GetAllLinks()
	if err != nil {
		log.Printf("API GetLinks error: %v", err)
		writeErrorJSON(w, "Failed to retrieve links", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(links)
}

// handleCreateLink creates a new link from the request body.
// CreateLink godoc
// @Summary      Create a link
// @Description  Create a new link
// @Tags         links
// @Accept       json
// @Produce      json
// @Param        link  body      Link  true  "Link payload"
// @Success      201
// @Failure      400  {string}  string  "Invalid request body"
// @Failure      500  {string}  string  "Failed to create link"
// @Router       /links [post]
func (s *Server) handleCreateLink(w http.ResponseWriter, r *http.Request) {
	var link Link
	if err := json.NewDecoder(r.Body).Decode(&link); err != nil {
		writeErrorJSON(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validateLink(link); err != nil {
		writeErrorJSON(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if err := s.store.CreateLink(link.Path, link.URL); err != nil {
		log.Printf("API CreateLink error: %v", err)
		// Check if it's a user-friendly error (like duplicate path)
		if strings.Contains(err.Error(), "already exists") {
			writeErrorJSON(w, err.Error(), http.StatusConflict)
			return
		}
		writeErrorJSON(w, "Failed to create link", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// handleUpdateLink updates an existing link.
// UpdateLink godoc
// @Summary      Update a link
// @Description  Update an existing link by ID
// @Tags         links
// @Accept       json
// @Produce      json
// @Param        id    path      int   true  "Link ID"
// @Param        link  body      Link  true  "Link payload"
// @Success      200
// @Failure      400  {string}  string  "Invalid request body"
// @Failure      500  {string}  string  "Failed to update link"
// @Router       /links/{id} [put]
func (s *Server) handleUpdateLink(w http.ResponseWriter, r *http.Request, id int64) {
	// Check if link exists first
	exists, err := s.store.LinkExists(id)
	if err != nil {
		log.Printf("API UpdateLink existence check error: %v", err)
		writeErrorJSON(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if !exists {
		writeErrorJSON(w, fmt.Sprintf("Link with id %d not found", id), http.StatusNotFound)
		return
	}

	var link Link
	if err := json.NewDecoder(r.Body).Decode(&link); err != nil {
		writeErrorJSON(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validateLink(link); err != nil {
		writeErrorJSON(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if err := s.store.UpdateLink(id, link.Path, link.URL); err != nil {
		log.Printf("API UpdateLink error: %v", err)
		// Check if it's a user-friendly error (like duplicate path)
		if strings.Contains(err.Error(), "already exists") {
			writeErrorJSON(w, err.Error(), http.StatusConflict)
			return
		}
		writeErrorJSON(w, "Failed to update link", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleDeleteLink deletes a link by its ID.
// DeleteLink godoc
// @Summary      Delete a link
// @Description  Delete a link by ID
// @Tags         links
// @Param        id  path  int  true  "Link ID"
// @Success      200
// @Failure      500  {string}  string  "Failed to delete link"
// @Router       /links/{id} [delete]
func (s *Server) handleDeleteLink(w http.ResponseWriter, r *http.Request, id int64) {
	if err := s.store.DeleteLink(id); err != nil {
		log.Printf("API DeleteLink error: %v", err)
		// Check if it's a "not found" error
		if strings.Contains(err.Error(), "not found") {
			writeErrorJSON(w, err.Error(), http.StatusNotFound)
			return
		}
		writeErrorJSON(w, "Failed to delete link", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// validateLink ensures the link payload has a valid path and HTTP/HTTPS URL.
func validateLink(link Link) error {
	// Validate path
	if err := validatePath(link.Path); err != nil {
		return err
	}

	// Validate URL
	if strings.TrimSpace(link.URL) == "" {
		return fmt.Errorf("url is required")
	}
	u, err := url.ParseRequestURI(link.URL)
	if err != nil {
		return fmt.Errorf("invalid url")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported url scheme")
	}
	if u.Host == "" {
		return fmt.Errorf("url host is required")
	}
	return nil
}

// validatePath ensures the path follows allowed format rules and isn't reserved.
func validatePath(path string) error {
	// Trim whitespace
	path = strings.TrimSpace(path)

	// Length constraints
	if len(path) == 0 {
		return fmt.Errorf("path is required")
	}
	if len(path) > 50 {
		return fmt.Errorf("path must be 50 characters or less")
	}

	// Format validation (alphanumeric, hyphens, underscores only)
	// Allow both uppercase and lowercase, but we'll normalize to lowercase in storage
	if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(path) {
		return fmt.Errorf("path can only contain letters, numbers, hyphens, and underscores")
	}

	// Check for reserved words (case-insensitive)
	pathLower := strings.ToLower(path)
	reserved := []string{"api", "swagger", "go", "favicon.ico", "robots.txt"}
	for _, word := range reserved {
		if pathLower == word {
			return fmt.Errorf("'%s' is a reserved path", path)
		}
	}

	return nil
}

// ErrorResponse represents a structured error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// writeErrorJSON writes a structured JSON error response.
func writeErrorJSON(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	response := ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	}
	json.NewEncoder(w).Encode(response)
}
