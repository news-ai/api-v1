package middleware

import (
	"net/http"
	"strings"

	"github.com/news-ai/api-v1/auth"
	apiControllers "github.com/news-ai/api-v1/controllers"
	"github.com/news-ai/api-v1/utils"
	// "github.com/news-ai/web/errors"

	"github.com/news-ai/web/errors"
)

func UpdateOrCreateUser(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	// Basic authentication
	apiKey, _, _ := r.BasicAuth()
	apiKeyValid := false
	if apiKey != "" {
		apiKeyValid = auth.BasicAuthLogin(w, r, apiKey)
	}

	email, err := auth.GetCurrentUserEmail(r)
	if err != nil && !strings.Contains(r.URL.Path, "/api/auth") && !strings.Contains(r.URL.Path, "/static") && !apiKeyValid {
		w.Header().Set("Content-Type", "application/json")
		errors.ReturnError(w, http.StatusUnauthorized, "Authentication Required", "Please login "+utils.APIURL+"/auth/google")
		return
	} else {
		if email != "" {
			apiControllers.AddUserToContext(r, email)
		}
	}

	next(w, r)
}
