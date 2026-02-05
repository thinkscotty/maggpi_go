package handlers

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/thinkscotty/maggpi_go/internal/database"
	"github.com/thinkscotty/maggpi_go/internal/models"
	"github.com/thinkscotty/maggpi_go/internal/scheduler"
	"github.com/thinkscotty/maggpi_go/internal/scraper"
)

// Handlers contains all HTTP handlers
type Handlers struct {
	db          *database.DB
	scheduler   *scheduler.Scheduler
	templates   map[string]*template.Template
	templateDir string
}

// New creates a new Handlers instance
func New(db *database.DB, sched *scheduler.Scheduler, templatesDir string) (*Handlers, error) {
	h := &Handlers{
		db:          db,
		scheduler:   sched,
		templates:   make(map[string]*template.Template),
		templateDir: templatesDir,
	}

	// Template functions
	funcMap := template.FuncMap{
		"json": func(v interface{}) template.JS {
			b, _ := json.Marshal(v)
			return template.JS(b)
		},
	}

	// Load each page template with base.html
	// Each page needs its own template set so "content" definitions don't overwrite each other
	pages := []string{"dashboard.html", "topics.html", "settings.html"}
	basePath := filepath.Join(templatesDir, "base.html")

	for _, page := range pages {
		pagePath := filepath.Join(templatesDir, page)
		tmpl, err := template.New("").Funcs(funcMap).ParseFiles(basePath, pagePath)
		if err != nil {
			return nil, err
		}
		h.templates[page] = tmpl
	}

	return h, nil
}

// render renders a template with data
func (h *Handlers) render(w http.ResponseWriter, tmpl string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	t, ok := h.templates[tmpl]
	if !ok {
		log.Printf("Template not found: %s", tmpl)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Execute the "base" template which will include the page's "content" block
	if err := t.ExecuteTemplate(w, "base", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// jsonResponse sends a JSON response
func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// jsonError sends an error JSON response
func jsonError(w http.ResponseWriter, status int, message string) {
	jsonResponse(w, status, models.APIResponse{
		Success: false,
		Error:   message,
	})
}

// Page handlers

// Dashboard renders the main dashboard page
func (h *Handlers) Dashboard(w http.ResponseWriter, r *http.Request) {
	settings, err := h.db.GetSettings()
	if err != nil {
		log.Printf("Error getting settings: %v", err)
		settings = &models.Settings{}
	}

	topics, err := h.db.GetTopicsWithStories(settings.StoriesPerTopic)
	if err != nil {
		log.Printf("Error getting topics: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":    "Dashboard",
		"Topics":   topics,
		"Settings": settings,
	}

	h.render(w, "dashboard.html", data)
}

// ManageTopics renders the topic management page
func (h *Handlers) ManageTopics(w http.ResponseWriter, r *http.Request) {
	settings, _ := h.db.GetSettings()
	topics, err := h.db.GetTopicsWithSources()
	if err != nil {
		log.Printf("Error getting topics: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":    "Manage Topics",
		"Topics":   topics,
		"Settings": settings,
	}

	h.render(w, "topics.html", data)
}

// Settings renders the settings page
func (h *Handlers) Settings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.db.GetSettings()
	if err != nil {
		log.Printf("Error getting settings: %v", err)
		settings = &models.Settings{}
	}

	data := map[string]interface{}{
		"Title":    "Settings",
		"Settings": settings,
	}

	h.render(w, "settings.html", data)
}

// API handlers for topics

// GetTopics returns all topics as JSON
func (h *Handlers) GetTopics(w http.ResponseWriter, r *http.Request) {
	topics, err := h.db.GetTopics()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, models.APIResponse{Success: true, Data: topics})
}

// CreateTopic creates a new topic
func (h *Handlers) CreateTopic(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		jsonError(w, http.StatusBadRequest, "Topic name is required")
		return
	}

	topic, err := h.db.CreateTopic(req.Name, req.Description)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Trigger source discovery in background
	go func() {
		if err := h.scheduler.DiscoverSources(topic.ID); err != nil {
			log.Printf("Error discovering sources for new topic: %v", err)
		}
	}()

	jsonResponse(w, http.StatusCreated, models.APIResponse{Success: true, Data: topic})
}

// UpdateTopic updates an existing topic
func (h *Handlers) UpdateTopic(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid topic ID")
		return
	}

	// Get existing topic to check if description changed
	existingTopic, err := h.db.GetTopic(id)
	if err != nil || existingTopic == nil {
		jsonError(w, http.StatusNotFound, "Topic not found")
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	descriptionChanged := existingTopic.Description != req.Description

	if err := h.db.UpdateTopic(id, req.Name, req.Description); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// If description changed, re-discover sources
	if descriptionChanged {
		go func() {
			if err := h.scheduler.DiscoverSources(id); err != nil {
				log.Printf("Error re-discovering sources: %v", err)
			}
		}()
	}

	jsonResponse(w, http.StatusOK, models.APIResponse{Success: true})
}

// DeleteTopic deletes a topic
func (h *Handlers) DeleteTopic(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid topic ID")
		return
	}

	if err := h.db.DeleteTopic(id); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, models.APIResponse{Success: true})
}

// ReorderTopics updates topic positions
func (h *Handlers) ReorderTopics(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TopicIDs []int64 `json:"topic_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.db.ReorderTopics(req.TopicIDs); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, models.APIResponse{Success: true})
}

// RefreshTopic manually triggers a topic refresh
func (h *Handlers) RefreshTopic(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid topic ID")
		return
	}

	go func() {
		if err := h.scheduler.RefreshTopic(id); err != nil {
			log.Printf("Error refreshing topic: %v", err)
		}
	}()

	jsonResponse(w, http.StatusOK, models.APIResponse{Success: true, Data: "Refresh started"})
}

// API handlers for sources

// AddSource adds a manual source to a topic
func (h *Handlers) AddSource(w http.ResponseWriter, r *http.Request) {
	topicID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid topic ID")
		return
	}

	var req struct {
		URL  string `json:"url"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := scraper.ValidateURL(req.URL); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	source, err := h.db.AddSource(topicID, req.URL, req.Name, true)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusCreated, models.APIResponse{Success: true, Data: source})
}

// DeleteSource removes a source
func (h *Handlers) DeleteSource(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "sourceId"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid source ID")
		return
	}

	if err := h.db.DeleteSource(id); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, models.APIResponse{Success: true})
}

// API handlers for settings

// GetSettings returns current settings
func (h *Handlers) GetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.db.GetSettings()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Don't expose the full API key
	if settings.GeminiAPIKey != "" {
		settings.GeminiAPIKey = "********" + settings.GeminiAPIKey[len(settings.GeminiAPIKey)-4:]
	}

	jsonResponse(w, http.StatusOK, models.APIResponse{Success: true, Data: settings})
}

// UpdateSettings updates application settings
func (h *Handlers) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req models.Settings
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get current settings to preserve API key if not changed
	current, _ := h.db.GetSettings()
	if current != nil && (req.GeminiAPIKey == "" || req.GeminiAPIKey[:8] == "********") {
		req.GeminiAPIKey = current.GeminiAPIKey
	}

	req.ID = 1
	if err := h.db.UpdateSettings(&req); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Update scheduler interval
	h.scheduler.UpdateInterval(req.RefreshIntervalMinutes)

	jsonResponse(w, http.StatusOK, models.APIResponse{Success: true})
}

// External API for client devices

// APIGetAllStories returns all topics with stories for external clients
func (h *Handlers) APIGetAllStories(w http.ResponseWriter, r *http.Request) {
	settings, _ := h.db.GetSettings()
	storiesPerTopic := 5
	if settings != nil {
		storiesPerTopic = settings.StoriesPerTopic
	}

	topics, err := h.db.GetTopicsWithStories(storiesPerTopic)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, models.APIResponse{Success: true, Data: topics})
}

// APIGetTopicStories returns stories for a specific topic
func (h *Handlers) APIGetTopicStories(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid topic ID")
		return
	}

	settings, _ := h.db.GetSettings()
	limit := 5
	if settings != nil {
		limit = settings.StoriesPerTopic
	}

	// Check for limit query param
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	topic, err := h.db.GetTopic(id)
	if err != nil || topic == nil {
		jsonError(w, http.StatusNotFound, "Topic not found")
		return
	}

	stories, err := h.db.GetStoriesForTopic(id, limit)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data: models.TopicWithStories{
			Topic:   *topic,
			Stories: stories,
		},
	})
}

// APIGetRefreshStatus returns refresh status for all topics
func (h *Handlers) APIGetRefreshStatus(w http.ResponseWriter, r *http.Request) {
	statuses, err := h.db.GetAllRefreshStatuses()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, models.APIResponse{Success: true, Data: statuses})
}
