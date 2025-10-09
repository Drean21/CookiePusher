package handler

import (
	"cookie-syncer/api/internal/store"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

// CreateUsersHandler handles bulk user creation (Admin only).
// POST /api/v1/admin/users
func CreateUsersHandler(db store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Count int    `json:"count"`
			Role  string `json:"role"`
		}
		// Set defaults
		payload.Count = 1
		payload.Role = "user"

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			RespondWithError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}
		if payload.Count <= 0 || payload.Count > 100 { // Limit bulk creation size
			RespondWithError(w, http.StatusBadRequest, "Count must be between 1 and 100")
			return
		}

		users, err := db.CreateUsers(payload.Count, payload.Role)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, "Could not create users")
			return
		}
		RespondWithJSON(w, http.StatusCreated, "Users created successfully", users)
	}
}

// DeleteUsersHandler handles bulk user deletion (Admin only).
// DELETE /api/v1/admin/users
func DeleteUsersHandler(db store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			UserIDs []int64  `json:"user_ids"`
			APIKeys []string `json:"api_keys"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			RespondWithError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}

		if len(payload.UserIDs) == 0 && len(payload.APIKeys) == 0 {
			RespondWithError(w, http.StatusBadRequest, "user_ids or api_keys must be provided")
			return
		}

		var totalDeleted int64
		if len(payload.UserIDs) > 0 {
			deleted, err := db.DeleteUsersByIDs(payload.UserIDs)
			if err != nil {
				RespondWithError(w, http.StatusInternalServerError, "Error deleting users by ID")
				return
			}
			totalDeleted += deleted
		}
		if len(payload.APIKeys) > 0 {
			deleted, err := db.DeleteUsersByAPIKeys(payload.APIKeys)
			if err != nil {
				RespondWithError(w, http.StatusInternalServerError, "Error deleting users by API Key")
				return
			}
			totalDeleted += deleted
		}
		
		RespondWithJSON(w, http.StatusOK, "Delete operation completed", map[string]int64{"total_deleted": totalDeleted})
	}
}

// RefreshSelfAPIKeyHandler allows a user to refresh their own API key.
// POST /api/v1/auth/refresh-key
func RefreshSelfAPIKeyHandler(db store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r.Context())
		if user == nil {
			RespondWithError(w, http.StatusInternalServerError, "Could not identify user")
			return
		}

		newAPIKey := uuid.New().String()
		if err := db.UpdateUserAPIKey(user.ID, newAPIKey); err != nil {
			RespondWithError(w, http.StatusInternalServerError, "Could not refresh API key")
			return
		}
		RespondWithJSON(w, http.StatusOK, "API key refreshed successfully", map[string]string{"new_api_key": newAPIKey})
	}
}

// AdminRefreshAPIKeysHandler allows an admin to refresh keys for multiple users.
// PUT /api/v1/admin/users/keys
func AdminRefreshAPIKeysHandler(db store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			UserIDs []int64 `json:"user_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			RespondWithError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}
		if len(payload.UserIDs) == 0 {
			RespondWithError(w, http.StatusBadRequest, "user_ids must be provided")
			return
		}

		refreshedUsers, err := db.RefreshUsersAPIKeys(payload.UserIDs)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, "Could not refresh user keys")
			return
		}
		RespondWithJSON(w, http.StatusOK, "Keys refreshed successfully", refreshedUsers)
	}
}
