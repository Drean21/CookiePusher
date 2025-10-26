package handler

import (
	"cookie-syncer/api/internal/store"
	"encoding/json"
	"net/http"

	"sort"
	"strings"

	"cookie-syncer/api/internal/model"

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

// GetSharableCookiesHandler handles fetching sharable cookies for a domain from the public pool.
// @Summary      Get sharable cookies by domain
// @Description  Retrieves all sharable cookies for a given domain from users who have opted into sharing.
// @Description  This endpoint is protected by a dedicated Pool Access Key (`x-pool-key` header), not a user's API key.
// @Description  By default, returns an array of strings, where each string is a user's cookies formatted as an HTTP 'Cookie' header.
// @Description  Use `?format=json` to get a structured JSON response, where each element contains the user's ID and their list of cookies.
// @Tags         Pool
// @Produce      json
// @Param        domain   path      string  true   "The domain to fetch cookies for"
// @Param        format   query     string  false  "Output format"  Enums(json)
// @Success      200      {object}  handler.APIResponse{data=[]string} "Default response: Array of HTTP Cookie header strings"
// @Success      200      {object}  handler.APIResponse{data=[]object{user_id=int,cookies=object}} "JSON response with `?format=json`"
// @Failure      401      {object}  handler.APIResponse "Unauthorized"
// @Failure      500      {object}  handler.APIResponse "Internal Server Error"
// @Security     PoolKeyAuth
// @Router       /pool/cookies/{domain} [get]
func GetSharableCookiesHandler(db store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		domain := chi.URLParam(r, "domain")
		allCookies, err := db.GetSharableCookiesByDomain(domain)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, "Could not fetch sharable cookies")
			return
		}

		// Group cookies by UserID
		cookiesByUser := make(map[int64][]*model.Cookie)
		for _, cookie := range allCookies {
			cookiesByUser[cookie.UserID] = append(cookiesByUser[cookie.UserID], cookie)
		}

		// Sort user IDs for consistent output order
		sortedUserIDs := make([]int64, 0, len(cookiesByUser))
		for userID := range cookiesByUser {
			sortedUserIDs = append(sortedUserIDs, userID)
		}
		sort.Slice(sortedUserIDs, func(i, j int) bool { return sortedUserIDs[i] < sortedUserIDs[j] })

		format := r.URL.Query().Get("format")
		if format == "json" {
			// JSON format: [{user_id: 1, cookies: {"domain": {"name": "value"}}}, ...]
			type userCookies struct {
				UserID  int64                        `json:"user_id"`
				Cookies map[string]map[string]string `json:"cookies"`
			}
			var result []userCookies
			for _, userID := range sortedUserIDs {
				domainMap := make(map[string]map[string]string)
				for _, cookie := range cookiesByUser[userID] {
					if _, ok := domainMap[cookie.Domain]; !ok {
						domainMap[cookie.Domain] = make(map[string]string)
					}
					domainMap[cookie.Domain][cookie.Name] = cookie.Value
				}

				result = append(result, userCookies{
					UserID:  userID,
					Cookies: domainMap,
				})
			}
			RespondWithJSON(w, http.StatusOK, "Successfully retrieved sharable cookies", result)
		} else {
			// Default format: ["cookie1=v1; cookie2=v2", "cookieA=vA; cookieB=vB"]
			var result []string
			for _, userID := range sortedUserIDs {
				var cookieParts []string
				for _, cookie := range cookiesByUser[userID] {
					cookieParts = append(cookieParts, cookie.Name+"="+cookie.Value)
				}
				result = append(result, strings.Join(cookieParts, "; "))
			}
			RespondWithJSON(w, http.StatusOK, "Successfully retrieved sharable cookies", result)
		}
	}
}
