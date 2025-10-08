package router

import (
	"cookie-syncer/api/internal/handler"
	"cookie-syncer/api/internal/store"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
)

// NewRouter creates and configures a new HTTP router using chi.
func NewRouter(db store.Store) *chi.Mux {
	r := chi.NewRouter()

	// A good base middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(Logger) // Our custom logger
	r.Use(middleware.Recoverer)

	// Set a timeout value on the request context (useful for databases and backend services)
	r.Use(middleware.Timeout(60 * time.Second))

	// Unauthenticated routes
	r.Get("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		handler.RespondWithJSON(w, http.StatusOK, "Service is healthy", nil)
	})

	// Authenticated routes group
	r.Group(func(r chi.Router) {
		r.Use(handler.AuthMiddleware(db))

		// Endpoint for authenticating and syncing cookies
		r.Post("/api/v1/sync", handler.SyncHandler(db))
		
		// Endpoint specifically for testing token validity
		r.Get("/api/v1/auth/test", func(w http.ResponseWriter, r *http.Request) {
			// If the request reaches here, the middleware has already validated the token.
			// We can also retrieve the user from context to be extra sure.
			user := handler.UserFromContext(r.Context())
			handler.RespondWithJSON(w, http.StatusOK, "Token is valid", map[string]interface{}{"user_id": user.ID, "role": user.Role})
		})
	})

	return r
}

// Logger is a custom middleware to log requests using zerolog.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		
		defer func() {
			log.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", ww.Status()).
				Dur("latency", time.Since(start)).
				Int("bytes", ww.BytesWritten()).
				Str("request_id", middleware.GetReqID(r.Context())).
				Msg("Request handled")
		}()
		
		next.ServeHTTP(ww, r)
	})
}

// PrintRoutes is a helper function to print all registered routes.
func PrintRoutes(r *chi.Mux) {
	fmt.Println("Registered routes:")
	walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		fmt.Printf("%-6s %s\n", method, route)
		return nil
	}
	if err := chi.Walk(r, walkFunc); err != nil {
		fmt.Printf("Error walking routes: %s\n", err.Error())
	}
}
