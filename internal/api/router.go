package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/thinkscotty/maggpi_go/internal/handlers"
)

// NewRouter creates and configures the HTTP router
func NewRouter(h *handlers.Handlers, staticDir string) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	// Serve static files
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	// Web UI routes
	r.Get("/", h.Dashboard)
	r.Get("/topics", h.ManageTopics)
	r.Get("/settings", h.Settings)

	// Internal API routes (for web UI)
	r.Route("/api", func(r chi.Router) {
		// Topics
		r.Get("/topics", h.GetTopics)
		r.Post("/topics", h.CreateTopic)
		r.Put("/topics/{id}", h.UpdateTopic)
		r.Delete("/topics/{id}", h.DeleteTopic)
		r.Post("/topics/reorder", h.ReorderTopics)
		r.Post("/topics/{id}/refresh", h.RefreshTopic)

		// Sources
		r.Post("/topics/{id}/sources", h.AddSource)
		r.Delete("/topics/{id}/sources/{sourceId}", h.DeleteSource)

		// Settings
		r.Get("/settings", h.GetSettings)
		r.Put("/settings", h.UpdateSettings)

		// Status
		r.Get("/status", h.APIGetRefreshStatus)
	})

	// External API routes (for client devices)
	r.Route("/v1", func(r chi.Router) {
		r.Get("/stories", h.APIGetAllStories)
		r.Get("/topics/{id}/stories", h.APIGetTopicStories)
		r.Get("/topics", h.GetTopics)
	})

	return r
}
