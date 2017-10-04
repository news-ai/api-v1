package auth

import (
	"errors"
	"net/http"
	"os"

	"github.com/julienschmidt/httprouter"

	"github.com/news-ai/api-v1/utils"
	"gopkg.in/boj/redistore.v1"
)

type User struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
	Hd            string `json:"hd"`
}

var store *redistore.RediStore

func SetupAuthStore() error {
	err := errors.New("")
	store, err = redistore.NewRediStore(10, "tcp", ":6379", "", []byte(os.Getenv("NEWSAI_SECRETKEY")))
	if err != nil {
		return err
	}
	googleOauthConfig.RedirectURL = utils.APIURL + "/auth/googlecallback"
	return nil
}

// Gets the email of the current user that is logged in
func GetCurrentUserEmail(r *http.Request) (string, error) {
	session, err := store.Get(r, "sess")
	if err != nil {
		return "", errors.New("No user logged in")
	}

	if session.Values["email"] == nil {
		return "", errors.New("No user logged in")
	}

	return session.Values["email"].(string), nil
}

// Gets the email of the current user that is logged in
func GetCurrentUserId(r *http.Request) (int64, error) {
	session, err := store.Get(r, "sess")
	if err != nil {
		return 0, errors.New("No user logged in")
	}

	if session.Values["id"] == nil {
		return 0, errors.New("No user logged in")
	}

	return session.Values["id"].(int64), nil
}

func LogoutHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	session, _ := store.Get(r, "sess")
	delete(session.Values, "state")
	delete(session.Values, "id")
	delete(session.Values, "email")
	session.Save(r, w)

	if r.URL.Query().Get("next") != "" {
		http.Redirect(w, r, r.URL.Query().Get("next"), 302)
		return
	}

	http.Redirect(w, r, "/api/auth", 302)
}
