package auth

import (
	"net/http"

	"github.com/news-ai/api-v1/controllers"

	"github.com/news-ai/web/utilities"
)

func BasicAuthLogin(w http.ResponseWriter, r *http.Request, apiKey string) bool {
	// Generate a random state that we identify the user with
	state := utilities.RandToken()

	// Save the session for each of the users
	session, _ := store.Get(r, "sess")
	session.Values["state"] = state
	session.Save(r, w)

	user, err := controllers.GetUserFromApiKey(apiKey)
	if err != nil {
		return false
	}

	session.Values["id"] = user.Id
	session.Values["email"] = user.Data.Email
	session.Save(r, w)

	return true
}

func BasicAuthLogout(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "sess")
	delete(session.Values, "state")
	delete(session.Values, "email")
	session.Save(r, w)
}
