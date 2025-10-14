package handler

import (
	"cookie-syncer/api/internal/store"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// CreateUsersHandler handles bulk user creation (Admin only).
// @Summary      Create one or more users
// @Description  Creates new users. The 'role' field must be either 'admin' or 'user'. The system enforces a singleton admin policy; this endpoint will fail if an admin user already exists and a new one is requested. Only accessible by admins.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        body body object{count=int,role=string} true "Request body"
// @Success      201  {object}  handler.APIResponse{data=[]model.User}
// @Failure      400  {object}  handler.APIResponse
// @Failure      403  {object}  handler.APIResponse
// @Failure      409  {object}  handler.APIResponse "Conflict if admin user already exists"
// @Failure      500  {object}  handler.APIResponse
// @Security     ApiKeyAuth
// @Router       /admin/users [post]
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
		if payload.Role != "admin" && payload.Role != "user" {
			RespondWithError(w, http.StatusBadRequest, "Role must be either 'admin' or 'user'")
			return
		}

		users, err := db.CreateUsers(payload.Count, payload.Role)
		if err != nil {
			if strings.Contains(err.Error(), "already exists") {
				RespondWithError(w, http.StatusConflict, "An admin user already exists.")
			} else {
				RespondWithError(w, http.StatusInternalServerError, "Could not create users")
			}
			return
		}
		RespondWithJSON(w, http.StatusCreated, "Users created successfully", users)
	}
}

// DeleteUsersHandler handles bulk user deletion (Admin only).
// @Summary      Delete one or more users
// @Description  Deletes users by their IDs or API keys. Only accessible by admins.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        body body object{user_ids=[]int,api_keys=[]string} true "User IDs or API Keys to delete"
// @Success      200  {object}  handler.APIResponse{data=object{total_deleted=int}}
// @Failure      400  {object}  handler.APIResponse
// @Failure      403  {object}  handler.APIResponse
// @Failure      500  {object}  handler.APIResponse
// @Security     ApiKeyAuth
// @Router       /admin/users [delete]
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
// @Summary      Refresh own API key
// @Description  Invalidates the current API key and returns a new one.
// @Tags         Auth
// @Produce      json
// @Success      200  {object}  handler.APIResponse{data=object{new_api_key=string}}
// @Failure      401  {object}  handler.APIResponse
// @Failure      500  {object}  handler.APIResponse
// @Security     ApiKeyAuth
// @Router       /auth/refresh-key [post]
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
// @Summary      Refresh API keys for users
// @Description  Generates new API keys for a list of user IDs. Only accessible by admins.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        body body object{user_ids=[]int} true "User IDs to refresh"
// @Success      200  {object}  handler.APIResponse{data=[]model.User}
// @Failure      400  {object}  handler.APIResponse
// @Failure      403  {object}  handler.APIResponse
// @Failure      500  {object}  handler.APIResponse
// @Security     ApiKeyAuth
// @Router       /admin/users/keys [put]
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

