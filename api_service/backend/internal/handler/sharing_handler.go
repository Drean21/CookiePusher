package handler

import (
	"cookie-syncer/api/internal/store"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// UpdateUserSettingsHandler handles updating a user's sharing settings.
// @Summary      Update user settings
// @Description  Updates settings for the authenticated user, such as enabling or disabling cookie sharing.
// @Tags         User
// @Accept       json
// @Produce      json
// @Param        body body object{sharing_enabled=bool} true "Settings payload"
// @Success      200  {object}  handler.APIResponse
// @Failure      400  {object}  handler.APIResponse
// @Failure      401  {object}  handler.APIResponse
// @Failure      500  {object}  handler.APIResponse
// @Security     ApiKeyAuth
// @Router       /user/settings [put]
func UpdateUserSettingsHandler(db store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r.Context())
		if user == nil {
			RespondWithError(w, http.StatusInternalServerError, "Could not identify user")
			return
		}

		var payload struct {
			SharingEnabled bool `json:"sharing_enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			RespondWithError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}

		if err := db.UpdateUserSharing(user.ID, payload.SharingEnabled); err != nil {
			RespondWithError(w, http.StatusInternalServerError, "Could not update user settings")
			return
		}

		RespondWithJSON(w, http.StatusOK, "User settings updated successfully", nil)
	}
}

// GetUserSettingsHandler handles fetching a user's sharing settings.
// @Summary      Get user settings
// @Description  Retrieves settings for the authenticated user, such as whether cookie sharing is enabled.
// @Tags         User
// @Produce      json
// @Success      200  {object}  handler.APIResponse{data=object{sharing_enabled=bool}}
// @Failure      401  {object}  handler.APIResponse
// @Failure      500  {object}  handler.APIResponse
// @Security     ApiKeyAuth
// @Router       /user/settings [get]
func GetUserSettingsHandler(db store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r.Context())
		if user == nil {
			RespondWithError(w, http.StatusInternalServerError, "Could not identify user")
			return
		}

		RespondWithJSON(w, http.StatusOK, "Successfully retrieved user settings", map[string]bool{
			"sharing_enabled": user.SharingEnabled,
		})
	}
}

// GetSharableCookiesHandler handles fetching sharable cookies for a domain.
// @Summary      [Admin] Get sharable cookies for a domain
// @Description  (Admin-only) Retrieves all cookies for a given domain that have been marked as "sharable" by users who have enabled sharing. By default, returns an HTTP header string. Use ?format=json to get structured JSON.
// @Tags         Admin
// @Produce      json
// @Param        domain   path      string  true   "Domain"
// @Param        format   query     string  false  "Output format"  Enums(json)
// @Success      200      {object}  handler.APIResponse{data=string}
// @Failure      401      {object}  handler.APIResponse "Unauthorized"
// @Failure      403      {object}  handler.APIResponse "Forbidden"
// @Failure      500      {object}  handler.APIResponse
// @Security     ApiKeyAuth
// @Router       /admin/pool/cookies/{domain} [get]
func GetSharableCookiesHandler(db store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		domain := chi.URLParam(r, "domain")
		cookies, err := db.GetSharableCookiesByDomain(domain)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, "Could not fetch sharable cookies")
			return
		}

		format := r.URL.Query().Get("format")
		if format == "json" {
			RespondWithJSON(w, http.StatusOK, "Successfully retrieved sharable cookies", cookies)
			return
		}

		// Default to HTTP Header string format
		var cookieParts []string
		for _, cookie := range cookies {
			cookieParts = append(cookieParts, cookie.Name+"="+cookie.Value)
		}
		cookieHeader := strings.Join(cookieParts, "; ")

		RespondWithJSON(w, http.StatusOK, "Successfully retrieved sharable cookies", cookieHeader)
	}
}
