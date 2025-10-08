package handler

import (
	"cookie-syncer/api/internal/model"
	"cookie-syncer/api/internal/store"
	"encoding/json"
	"net/http"
)

// SyncHandler handles the main data synchronization endpoint.
// It receives cookie data from the extension, processes it, and returns the full updated state.
//
// POST /api/v1/sync
func SyncHandler(db store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// The user is authenticated by the middleware at this point.
		user := UserFromContext(r.Context())
		if user == nil {
			// This should theoretically not happen if middleware is set up correctly
			RespondWithError(w, http.StatusInternalServerError, "Could not identify user")
			return
		}

		// 1. Decode JSON body into a slice of model.Cookie
		var cookiesToSync []*model.Cookie
		if err := json.NewDecoder(r.Body).Decode(&cookiesToSync); err != nil {
			RespondWithError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}

		// 2. Call db.SyncCookies to persist the data
		if err := db.SyncCookies(user.ID, cookiesToSync); err != nil {
			RespondWithError(w, http.StatusInternalServerError, "Could not sync cookies to database")
			return
		}

		// 3. Fetch the latest full cookie list from the DB
		latestCookies, err := db.GetCookiesByUserID(user.ID)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, "Could not fetch latest cookies")
			return
		}

		// 4. Encode the full list and return as JSON response
		RespondWithJSON(w, http.StatusOK, "Sync successful", latestCookies)
	}
}
