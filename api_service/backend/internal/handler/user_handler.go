package handler

import (
	"cookie-syncer/api/internal/config"
	"cookie-syncer/api/internal/model"
	"cookie-syncer/api/internal/store"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// AdminUserResponse is a specific view of the User model for admin responses,
// which includes the API key.
type AdminUserResponse struct {
	ID             int64      `json:"id"`
	APIKey         string     `json:"api_key"`
	Remark         *string    `json:"remark,omitempty"`
	SharingEnabled bool       `json:"sharing_enabled"`
	LastSyncedAt   *time.Time `json:"last_synced_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func toAdminUserResponse(user *model.User) AdminUserResponse {
	return AdminUserResponse{
		ID:             user.ID,
		APIKey:         user.APIKey,
		Remark:         user.Remark,
		SharingEnabled: user.SharingEnabled,
		LastSyncedAt:   user.LastSyncedAt,
		CreatedAt:      user.CreatedAt,
		UpdatedAt:      user.UpdatedAt,
	}
}

func toAdminUserResponses(users []*model.User) []AdminUserResponse {
	responses := make([]AdminUserResponse, len(users))
	for i, user := range users {
		responses[i] = toAdminUserResponse(user)
	}
	return responses
}

// AdminCreateUsersHandler handles bulk user creation by an admin.
// @Summary      [Admin] Create one or more users
// @Description  Creates new users based on the request body array. Each object in the array can specify a remark.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        body body []object{remark=string} true "Array of users to create. Can be empty to create one default user."
// @Success      201  {object}  handler.APIResponse{data=[]handler.AdminUserResponse} "Returns an array of created users including their new API keys."
// @Failure      400  {object}  handler.APIResponse
// @Failure      403  {object}  handler.APIResponse
// @Failure      500  {object}  handler.APIResponse
// @Security     AdminKeyAuth
// @Router       /admin/users [post]
func AdminCreateUsersHandler(db store.Store, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var usersToCreate []struct {
			Remark *string `json:"remark"`
		}

		if err := json.NewDecoder(r.Body).Decode(&usersToCreate); err != nil {
			// Handle case where body is empty, create one default user
			if err.Error() == "EOF" {
				usersToCreate = []struct {
					Remark *string `json:"remark"`
				}{{Remark: nil}}
			} else {
				RespondWithError(w, http.StatusBadRequest, "Invalid JSON body: "+err.Error())
				return
			}
		}

		var remarks []string
		for _, u := range usersToCreate {
			if u.Remark != nil {
				remarks = append(remarks, *u.Remark)
			} else {
				remarks = append(remarks, "") // Use empty string for nil remark
			}
		}

		if len(remarks) == 0 {
			remarks = append(remarks, "Default user")
		}

		createdUsers, err := db.CreateUsers(remarks)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, "Could not create users: "+err.Error())
			return
		}

		RespondWithJSON(w, http.StatusCreated, "Users created successfully", toAdminUserResponses(createdUsers))
	}
}

// AdminUpdateUserHandler handles updating a user's details by ID.
// @Summary      [Admin] Update user by ID
// @Description  Updates a user's details, such as their remark.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "User ID"
// @Param        body body      object{remark=string} true "User details to update"
// @Success      200  {object}  handler.APIResponse
// @Failure      400  {object}  handler.APIResponse
// @Failure      403  {object}  handler.APIResponse
// @Failure      500  {object}  handler.APIResponse
// @Security     AdminKeyAuth
// @Router       /admin/users/{id} [put]
func AdminUpdateUserHandler(db store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			RespondWithError(w, http.StatusBadRequest, "Invalid user ID")
			return
		}

		var payload struct {
			Remark *string `json:"remark"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			RespondWithError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}

		if err := db.UpdateUserRemark(id, payload.Remark); err != nil {
			RespondWithError(w, http.StatusInternalServerError, "Could not update user: "+err.Error())
			return
		}

		RespondWithJSON(w, http.StatusOK, "User updated successfully", nil)
	}
}

// AdminUpdateUserByAPIKeyHandler handles updating a user's details by API key.
// @Summary      [Admin] Update user by API key
// @Description  Updates a user's details, such as their remark, by finding them via API key.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        apiKey path      string true  "User API Key"
// @Param        body   body      object{remark=string} true "User details to update"
// @Success      200    {object}  handler.APIResponse
// @Failure      400    {object}  handler.APIResponse
// @Failure      403    {object}  handler.APIResponse
// @Failure      500    {object}  handler.APIResponse
// @Security     AdminKeyAuth
// @Router       /admin/users/by-key/{apiKey} [put]
func AdminUpdateUserByAPIKeyHandler(db store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := chi.URLParam(r, "apiKey")

		var payload struct {
			Remark *string `json:"remark"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			RespondWithError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}

		if err := db.UpdateUserRemarkByAPIKey(apiKey, payload.Remark); err != nil {
			RespondWithError(w, http.StatusInternalServerError, "Could not update user: "+err.Error())
			return
		}

		RespondWithJSON(w, http.StatusOK, "User updated successfully", nil)
	}
}

// AdminRefreshUserAPIKeyHandler handles refreshing a user's API key by ID.
// @Summary      [Admin] Refresh user API key by ID
// @Description  Generates a new API key for the specified user and returns the full updated user object.
// @Tags         Admin
// @Produce      json
// @Param        id   path      int  true  "User ID"
// @Success      200  {object}  handler.APIResponse{data=handler.AdminUserResponse}
// @Failure      400  {object}  handler.APIResponse
// @Failure      403  {object}  handler.APIResponse
// @Failure      500  {object}  handler.APIResponse
// @Security     AdminKeyAuth
// @Router       /admin/users/{id}/refresh-key [post]
func AdminRefreshUserAPIKeyHandler(db store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			RespondWithError(w, http.StatusBadRequest, "Invalid user ID")
			return
		}

		updatedUser, err := db.AdminUpdateUserAPIKey(id)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, "Could not refresh API key: "+err.Error())
			return
		}

		RespondWithJSON(w, http.StatusOK, "API key refreshed successfully", toAdminUserResponse(updatedUser))
	}
}

// AdminRefreshUserAPIKeyByAPIKeyHandler handles refreshing a user's API key by API key.
// @Summary      [Admin] Refresh user API key by API key
// @Description  Generates a new API key for the specified user and returns the full updated user object.
// @Tags         Admin
// @Produce      json
// @Param        apiKey path      string true  "User API Key"
// @Success      200    {object}  handler.APIResponse{data=handler.AdminUserResponse}
// @Failure      400    {object}  handler.APIResponse
// @Failure      403    {object}  handler.APIResponse
// @Failure      500    {object}  handler.APIResponse
// @Security     AdminKeyAuth
// @Router       /admin/users/by-key/{apiKey}/refresh-key [post]
func AdminRefreshUserAPIKeyByAPIKeyHandler(db store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := chi.URLParam(r, "apiKey")

		updatedUser, err := db.AdminUpdateUserAPIKeyByAPIKey(apiKey)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, "Could not refresh API key: "+err.Error())
			return
		}

		RespondWithJSON(w, http.StatusOK, "API key refreshed successfully", toAdminUserResponse(updatedUser))
	}
}
