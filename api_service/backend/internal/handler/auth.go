package handler

import (
	"context"
	"cookie-syncer/api/internal/model"
	"cookie-syncer/api/internal/store"
	"log"
	"net/http"
)

// contextKey is a custom type to avoid key collisions in context.
type contextKey string

const userContextKey = contextKey("user")

// AuthMiddleware creates a middleware to handle API key authentication.
func AuthMiddleware(db store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("x-api-key")
			if apiKey == "" {
				RespondWithError(w, http.StatusUnauthorized, "x-api-key header required")
				return
			}

			user, err := db.GetUserByAPIKey(apiKey)
			if err != nil {
				// Add critical logging for debugging authentication errors
				log.Printf("[Auth Failed] Middleware rejected request. Reason: %v. Received API Key: '%s'", err, apiKey)
				RespondWithError(w, http.StatusUnauthorized, "Invalid API Key")
				return
			}

			// Store user in context to pass to the next handler
			ctx := context.WithValue(r.Context(), userContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserFromContext retrieves the user from the request context.
// Returns nil if user is not found.
func UserFromContext(ctx context.Context) *model.User {
	user, ok := ctx.Value(userContextKey).(*model.User)
	if !ok {
		return nil
	}
	return user
}
