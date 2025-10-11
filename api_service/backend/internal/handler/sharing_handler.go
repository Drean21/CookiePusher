package handler

import (
	"cookie-syncer/api/internal/store"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// UpdateUserSettingsHandler handles updating a user's sharing settings.
// PUT /api/v1/user/settings
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

// GetSharableCookiesHandler handles fetching sharable cookies for a domain.
// GET /api/v1/pool/cookies/{domain}
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
