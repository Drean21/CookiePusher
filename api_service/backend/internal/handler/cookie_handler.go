package handler

import (
	"cookie-syncer/api/internal/store"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// GetAllCookiesHandler handles the request to get all cookies for a user, grouped by domain.
// GET /api/v1/cookies/all
func GetAllCookiesHandler(db store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r.Context())
		if user == nil {
			RespondWithError(w, http.StatusInternalServerError, "Could not identify user")
			return
		}

		allCookies, err := db.GetCookiesByUserID(user.ID)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, "Could not fetch cookies")
			return
		}
		
		format := r.URL.Query().Get("format")
		if format == "json" {
			// Group cookies by domain -> { name: value }
			groupedCookies := make(map[string]map[string]string)
			for _, cookie := range allCookies {
				domain := cookie.Domain
				if _, ok := groupedCookies[domain]; !ok {
					groupedCookies[domain] = make(map[string]string)
				}
				groupedCookies[domain][cookie.Name] = cookie.Value
			}
			RespondWithJSON(w, http.StatusOK, "Successfully retrieved all cookies", groupedCookies)
			return
		}

		// Default to HTTP Header string format, grouped by domain
		groupedCookieStrings := make(map[string]string)
		tempGroup := make(map[string][]string) // domain -> ["key=val", "key2=val2"]
		for _, cookie := range allCookies {
			tempGroup[cookie.Domain] = append(tempGroup[cookie.Domain], cookie.Name+"="+cookie.Value)
		}
		for domain, parts := range tempGroup {
			groupedCookieStrings[domain] = strings.Join(parts, "; ")
		}

		RespondWithJSON(w, http.StatusOK, "Successfully retrieved all cookies", groupedCookieStrings)
	}
}

// GetDomainCookiesHandler handles the request to get all cookies for a specific domain.
// It defaults to returning a string formatted for HTTP headers.
// Use ?format=json to get the full cookie objects.
// GET /api/v1/cookies/{domain}
func GetDomainCookiesHandler(db store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r.Context())
		if user == nil {
			RespondWithError(w, http.StatusInternalServerError, "Could not identify user")
			return
		}

		domain := chi.URLParam(r, "domain")
		cookies, err := db.GetCookiesByDomain(user.ID, domain)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, "Could not fetch cookies for domain")
			return
		}

		format := r.URL.Query().Get("format")
		if format == "json" {
			RespondWithJSON(w, http.StatusOK, "Successfully retrieved cookies for domain", cookies)
			return
		}

		// Default to HTTP Header string format
		var cookieParts []string
		for _, cookie := range cookies {
			cookieParts = append(cookieParts, cookie.Name+"="+cookie.Value)
		}
		cookieHeader := strings.Join(cookieParts, "; ")

		RespondWithJSON(w, http.StatusOK, "Successfully retrieved cookies for domain", cookieHeader)
	}
}

// GetCookieValueHandler handles the request to get the raw value of a specific cookie.
// The value is returned inside the 'data' field of the standard API response.
// GET /api/v1/cookies/{domain}/{name}
func GetCookieValueHandler(db store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r.Context())
		if user == nil {
			RespondWithError(w, http.StatusInternalServerError, "Could not identify user")
			return
		}

		domain := chi.URLParam(r, "domain")
		name := chi.URLParam(r, "name")

		cookie, err := db.GetCookieByName(user.ID, domain, name)
		if err != nil {
			if err.Error() == "cookie not found" {
				RespondWithError(w, http.StatusNotFound, "Cookie not found")
			} else {
				RespondWithError(w, http.StatusInternalServerError, "Internal server error")
			}
			return
		}

		RespondWithJSON(w, http.StatusOK, "Successfully retrieved cookie value", cookie.Value)
	}
}
