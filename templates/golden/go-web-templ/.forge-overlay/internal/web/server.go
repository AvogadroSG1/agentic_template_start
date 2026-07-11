package web

import (
	"embed"
	"io/fs"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

//go:embed static
var staticFiles embed.FS

// NewServer wires the walking skeleton: the page shell, the health
// contract consumed by HTMX, and the locally vendored static assets.
func NewServer(title string, logger *log.Logger) http.Handler {
	router := chi.NewRouter()

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := Index(title, "checking…").Render(r.Context(), w); err != nil {
			logger.Printf("render index: %v", err)
		}
	})

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		logger.Println("health endpoint served")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	router.Get("/health/fragment", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := StatusFragment("ok").Render(r.Context(), w); err != nil {
			logger.Printf("render status fragment: %v", err)
		}
	})

	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}
	router.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	return router
}
