package router

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/essajiwa/hooklab/internal/app/handler"
	"github.com/essajiwa/hooklab/internal/app/service"
)

// NewServer creates and configures the HTTP server with all routes.
// It registers webhook handlers, API endpoints, and serves static files from the web directory.
func NewServer(app *service.App, port int) (*http.Server, error) {
	r := chi.NewRouter()

	r.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		handler.WebhookHandler(app, w, r)
	})
	r.HandleFunc("/webhook/*", func(w http.ResponseWriter, r *http.Request) {
		handler.WebhookHandler(app, w, r)
	})
	r.Get("/api/events", func(w http.ResponseWriter, r *http.Request) {
		handler.EventsHandler(app, w, r)
	})
	r.Get("/api/stream", func(w http.ResponseWriter, r *http.Request) {
		handler.EventsStreamHandler(app, w, r)
	})
	r.HandleFunc("/api/response", func(w http.ResponseWriter, r *http.Request) {
		handler.ResponseHandler(app, w, r)
	})
	r.HandleFunc("/api/response/", func(w http.ResponseWriter, r *http.Request) {
		handler.ResponseHandler(app, w, r)
	})
	r.HandleFunc("/api/rules", func(w http.ResponseWriter, r *http.Request) {
		handler.RulesHandler(app, w, r)
	})
	r.Get("/api/keys", func(w http.ResponseWriter, r *http.Request) {
		handler.KeysHandler(app, w, r)
	})

	// Serve rules.html for /rules
	r.Get("/rules", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/rules.html")
	})

	r.Handle("/", http.FileServer(http.Dir("web")))

	server := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: r}
	return server, nil
}