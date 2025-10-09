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
				// Check if the error is specifically "user not found"
				if err.Error() == "user not found" {
					log.Printf("[Auth Failed] Middleware rejected request. Reason: user not found. Received API Key: '%s'", apiKey)
					RespondWithError(w, http.StatusUnauthorized, "Invalid API Key")
				} else {
					// For all other errors (like database locked), it's a server-side issue.
					log.Printf("[Auth Error] Middleware encountered a database error. Reason: %v. Received API Key: '%s'", err, apiKey)
					RespondWithError(w, http.StatusInternalServerError, "Internal Server Error during authentication")
				}
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

// AdminOnlyMiddleware is a middleware to ensure only users with the 'admin' role can proceed.
// It must be used AFTER the AuthMiddleware.
func AdminOnlyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r.Context())
		if user == nil {
			// This should not happen if AuthMiddleware is used before this.
			RespondWithError(w, http.StatusInternalServerError, "Could not identify user")
			return
		}

		if user.Role != "admin" {
			RespondWithError(w, http.StatusForbidden, "Forbidden: Administrator access required")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// AuthTestHandler is a simple handler to confirm that a token is valid.
func AuthTestHandler(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	RespondWithJSON(w, http.StatusOK, "Token is valid", map[string]interface{}{"user_id": user.ID, "role": user.Role})
}
