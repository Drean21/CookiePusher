package handler

import (
	"cookie-syncer/api/internal/store"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// GetAllCookiesHandler handles the request to get all cookies for a user.
// @Summary      Get all cookies
// @Description  Retrieves all cookies for the authenticated user. By default, groups them by domain and returns them as HTTP header strings. Use ?format=json to get structured JSON.
// @Tags         Cookies
// @Produce      json
// @Param        format query     string  false  "Output format"  Enums(json)
// @Success      200    {object}  handler.APIResponse{data=object}
// @Failure      401    {object}  handler.APIResponse
// @Failure      500    {object}  handler.APIResponse
// @Security     ApiKeyAuth
// @Router       /cookies/all [get]
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
// @Summary      Get cookies for a domain
// @Description  Retrieves cookies for a specific domain. By default, returns an HTTP header string. Use ?format=json to get structured JSON.
// @Tags         Cookies
// @Produce      json
// @Param        domain   path      string  true   "Domain"
// @Param        format   query     string  false  "Output format"  Enums(json)
// @Success      200      {object}  handler.APIResponse{data=string}
// @Failure      401      {object}  handler.APIResponse
// @Failure      500      {object}  handler.APIResponse
// @Security     ApiKeyAuth
// @Router       /cookies/{domain} [get]
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
// @Summary      Get a single cookie's value
// @Description  Retrieves the raw value of a specific cookie, returned in the 'data' field.
// @Tags         Cookies
// @Produce      json
// @Param        domain   path      string  true  "Domain"
// @Param        name     path      string  true  "Cookie Name"
// @Success      200      {object}  handler.APIResponse{data=string}
// @Failure      401      {object}  handler.APIResponse
// @Failure      404      {object}  handler.APIResponse
// @Failure      500      {object}  handler.APIResponse
// @Security     ApiKeyAuth
// @Router       /cookies/{domain}/{name} [get]
func GetCookieValueHandler(db store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r.Context())
		if user == nil {
			RespondWithError(w, http.StatusInternalServerError, "Could not identify user")
			return
		}

		domain := chi.URLParam(r, "domain")
		name := chi.URLParam(r, "name")

		// With the new storage model, it's more efficient to get all cookies for the domain
		// and then filter by name in the application.
		cookies, err := db.GetCookiesByDomain(user.ID, domain)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, "Could not retrieve cookies for domain")
			return
		}

		for _, cookie := range cookies {
			if cookie.Name == name {
				RespondWithJSON(w, http.StatusOK, "Successfully retrieved cookie value", cookie.Value)
				return
			}
		}

		RespondWithError(w, http.StatusNotFound, "Cookie not found")
	}
}
