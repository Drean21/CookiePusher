package router

import (
	"cookie-syncer/api/internal/config"
	"cookie-syncer/api/internal/handler"
	"cookie-syncer/api/internal/store"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
	httpSwagger "github.com/swaggo/http-swagger"
)

// NewRouter creates and configures a new HTTP router using chi.
func NewRouter(db store.Store, locker *handler.UserLockManager, cfg *config.Config) *chi.Mux {
	r := chi.NewRouter()

	// A good base middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(Logger) // Our custom logger
	r.Use(middleware.Recoverer)

	// Set a timeout value on the request context (useful for databases and backend services)
	r.Use(middleware.Timeout(60 * time.Second))

	// Swagger documentation
	r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
	})
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	// Unauthenticated routes
	r.Get("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		handler.RespondWithJSON(w, http.StatusOK, "Service is healthy", nil)
	})

	// Authenticated routes group for regular users
	r.Group(func(r chi.Router) {
		r.Use(handler.AuthMiddleware(db))

		r.Post("/api/v1/sync", handler.SyncHandler(db, locker))
		r.Get("/api/v1/auth/test", handler.AuthTestHandler)
		r.Get("/api/v1/cookies/all", handler.GetAllCookiesHandler(db))
		r.Get("/api/v1/cookies/{domain}", handler.GetDomainCookiesHandler(db))
		r.Get("/api/v1/cookies/{domain}/{name}", handler.GetCookieValueHandler(db))
		r.Get("/api/v1/user/settings", handler.GetUserSettingsHandler(db))
		r.Put("/api/v1/user/settings", handler.UpdateUserSettingsHandler(db))
	})

	// Pool API for shared cookies, protected by a separate key
	r.Group(func(r chi.Router) {
		// This middleware will check for the X-Pool-Key header
		r.Use(handler.PoolKeyAuthMiddleware(cfg.PoolAccessKey))
		r.Get("/api/v1/pool/cookies/{domain}", handler.GetSharableCookiesHandler(db))
	})

	// Admin-only routes group, protected by a separate key
	r.Group(func(r chi.Router) {
		r.Use(handler.AdminKeyAuthMiddleware(cfg.AdminKey))

		r.Post("/api/v1/admin/users", handler.AdminCreateUsersHandler(db, cfg))
		r.Put("/api/v1/admin/users/{id}", handler.AdminUpdateUserHandler(db))
		r.Put("/api/v1/admin/users/by-key/{apiKey}", handler.AdminUpdateUserByAPIKeyHandler(db))
		r.Post("/api/v1/admin/users/{id}/refresh-key", handler.AdminRefreshUserAPIKeyHandler(db))
		r.Post("/api/v1/admin/users/by-key/{apiKey}/refresh-key", handler.AdminRefreshUserAPIKeyByAPIKeyHandler(db))
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
