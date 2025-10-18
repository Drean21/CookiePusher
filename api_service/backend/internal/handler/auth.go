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

// PoolKeyAuthMiddleware returns a middleware that checks for a valid pool access key.
func PoolKeyAuthMiddleware(expectedKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if expectedKey == "" {
				log.Printf("[Auth Error] Pool access key is not configured on the server. Access to pool denied.")
				RespondWithError(w, http.StatusInternalServerError, "Pool access is not configured")
				return
			}

			poolKey := r.Header.Get("x-pool-key")
			if poolKey == "" {
				RespondWithError(w, http.StatusUnauthorized, "x-pool-key header required")
				return
			}

			if poolKey != expectedKey {
				log.Printf("[Auth Failed] Pool middleware rejected request. Reason: invalid pool key.")
				RespondWithError(w, http.StatusUnauthorized, "Invalid Pool Key")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// AdminKeyAuthMiddleware returns a middleware that checks for a valid admin key.
func AdminKeyAuthMiddleware(expectedKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if expectedKey == "" {
				log.Printf("[Auth Error] Admin key is not configured on the server. Access to admin endpoints denied.")
				RespondWithError(w, http.StatusInternalServerError, "Admin access is not configured")
				return
			}

			adminKey := r.Header.Get("x-admin-key")
			if adminKey == "" {
				RespondWithError(w, http.StatusUnauthorized, "x-admin-key header required")
				return
			}

			if adminKey != expectedKey {
				log.Printf("[Auth Failed] Admin middleware rejected request. Reason: invalid admin key.")
				RespondWithError(w, http.StatusForbidden, "Forbidden: Invalid Admin Key")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// AuthTestHandler is a simple handler to confirm that a token is valid.
// @Summary      Test API Key
// @Description  A simple endpoint to check if the provided API key in the `x-api-key` header is valid and associated with a user.
// @Tags         Auth
// @Produce      json
// @Success      200  {object}  handler.APIResponse{data=object{user_id=int,role=string}}
// @Failure      401  {object}  handler.APIResponse
// @Security     ApiKeyAuth
// @Router       /auth/test [get]
func AuthTestHandler(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	RespondWithJSON(w, http.StatusOK, "Token is valid", map[string]interface{}{"user_id": user.ID})
}
