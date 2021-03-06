package auth

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"golang.org/x/net/context"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/julienschmidt/httprouter"

	apiControllers "github.com/news-ai/api-v1/controllers"
	apiModels "github.com/news-ai/api-v1/models"

	tabulaeControllers "github.com/news-ai/tabulae-v1/controllers"
	"github.com/news-ai/tabulae-v1/emails"

	"github.com/news-ai/web/utilities"
)

var (
	googleOauthConfig = &oauth2.Config{
		RedirectURL:  "https://tabulae.newsai.org/api/auth/googlecallback",
		ClientID:     os.Getenv("GOOGLEAUTHKEY"),
		ClientSecret: os.Getenv("GOOGLEAUTHSECRET"),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/userinfo.email",
		},
		Endpoint: google.Endpoint,
	}

	gmailOauthConfig = &oauth2.Config{
		RedirectURL:  "https://tabulae.newsai.org/api/auth/googlecallback",
		ClientID:     os.Getenv("GOOGLEAUTHKEY"),
		ClientSecret: os.Getenv("GOOGLEAUTHSECRET"),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/gmail.readonly",
			"https://www.googleapis.com/auth/gmail.compose",
			"https://www.googleapis.com/auth/gmail.send",
		},
		Endpoint: google.Endpoint,
	}
)

// Handler to redirect user to the Google OAuth2 page
func GoogleLoginHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Generate a random state that we identify the user with
	state := utilities.RandToken()

	// Save the session for each of the users
	session, err := store.Get(r, "sess")
	if err != nil {
		log.Printf("%v", err)
	}

	session.Values["state"] = state
	session.Values["gmail"] = "no"
	session.Values["gmail_email"] = ""

	if r.URL.Query().Get("next") != "" {
		session.Values["next"] = r.URL.Query().Get("next")
	}

	err = session.Save(r, w)
	if err != nil {
		log.Printf("%v", err)
	}

	// Redirect the user to the login page
	url := googleOauthConfig.AuthCodeURL(state)
	http.Redirect(w, r, url, 302)
	return
}

// Handler to redirect user to the Google OAuth2 page
func RemoveGmailHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Make sure the user has been logged in when at gmail auth
	user, err := apiControllers.GetCurrentUser(r)
	if err != nil {
		log.Printf("%v", err)
		fmt.Fprintln(w, "user not logged in")
		return
	}

	user.Data.Gmail = false
	// apiControllers.SaveUser(c, r, &user)

	if r.URL.Query().Get("next") != "" {
		returnURL := r.URL.Query().Get("next")
		if err != nil {
			http.Redirect(w, r, returnURL, 302)
			return
		}
	}

	http.Redirect(w, r, "https://tabulae.newsai.co/settings", 302)
	return
}

// Handler to redirect user to the Google OAuth2 page
func GmailLoginHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Make sure the user has been logged in when at gmail auth
	user, err := apiControllers.GetCurrentUser(r)
	if err != nil {
		log.Printf("%v", err)
		fmt.Fprintln(w, "user not logged in")
		return
	}

	// Generate a random state that we identify the user with
	state := utilities.RandToken()

	// Save the session for each of the users
	session, err := store.Get(r, "sess")
	if err != nil {
		log.Printf("%v", err)
	}

	session.Values["state"] = state
	session.Values["gmail"] = "yes"
	session.Values["gmail_email"] = user.Data.Email

	if r.URL.Query().Get("next") != "" {
		session.Values["next"] = r.URL.Query().Get("next")
	}

	err = session.Save(r, w)
	if err != nil {
		log.Printf("%v", err)
	}

	// Redirect the user to the login page
	url := gmailOauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, 302)
}

// Handler to get information when callback comes back from Google
func GoogleCallbackHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	session, err := store.Get(r, "sess")
	if err != nil {
		log.Printf("%v", err)
		fmt.Fprintln(w, "aborted")
		return
	}

	if r.URL.Query().Get("state") != session.Values["state"] {
		log.Printf("%v", "no state match; possible csrf OR cookies not enabled")
		fmt.Fprintln(w, "no state match; possible csrf OR cookies not enabled")
		return
	}

	ctx := context.Background()
	tkn, err := googleOauthConfig.Exchange(ctx, r.URL.Query().Get("code"))

	if err != nil {
		log.Printf("%v", "there was an issue getting your token")
		fmt.Fprintln(w, "there was an issue getting your token")
		return
	}

	if !tkn.Valid() {
		log.Printf("%v", "retreived invalid token")
		fmt.Fprintln(w, "retreived invalid token")
		return
	}

	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?alt=json&access_token=" + tkn.AccessToken)
	if err != nil {
		log.Printf("%v", err)
		fmt.Fprintln(w, err.Error())
		return
	}
	defer resp.Body.Close()

	// Decode JSON from Google
	decoder := json.NewDecoder(resp.Body)
	var googleUser User
	err = decoder.Decode(&googleUser)
	if err != nil {
		log.Printf("%v", err)
		fmt.Fprintln(w, err.Error())
		return
	}

	newUser := apiModels.User{}
	newUser.Email = googleUser.Email
	newUser.GoogleId = googleUser.ID
	newUser.FirstName = googleUser.GivenName
	newUser.LastName = googleUser.FamilyName
	newUser.EmailConfirmed = true
	newUser.IsActive = false

	newUser.TokenType = tkn.TokenType
	newUser.GoogleExpiresIn = tkn.Expiry
	newUser.RefreshToken = tkn.RefreshToken
	newUser.AccessToken = tkn.AccessToken
	newUser.GoogleCode = r.URL.Query().Get("code")
	if session.Values["gmail"] == "yes" {
		newUser.Gmail = true
		newUser.Outlook = false
		newUser.ExternalEmail = false

		if session.Values["gmail_email"].(string) != googleUser.Email {
			log.Printf("%v", "Tried to login with email "+googleUser.Email+" for user "+session.Values["gmail_email"].(string))
			http.Redirect(w, r, "https://tabulae.newsai.co/settings", 302)
			return
		}
	}

	user, _, _ := tabulaeControllers.RegisterUser(r, newUser)

	session.Values["email"] = googleUser.Email
	session.Values["id"] = newUser.Id
	session.Save(r, w)

	if user.Data.IsActive {
		if session.Values["next"] != nil {
			returnURL := session.Values["next"].(string)
			u, err := url.Parse(returnURL)
			if err != nil {
				http.Redirect(w, r, returnURL, 302)
				return
			}

			if user.Data.LastLoggedIn.IsZero() {
				q := u.Query()
				q.Set("firstTimeUser", "true")
				u.RawQuery = q.Encode()

				err = emails.AddUserToTabulaeTrialList(user.Data)
				if err != nil {
					// Redirect user back to login page
					log.Printf("%v", "Welcome email was not sent for "+user.Data.Email)
					log.Printf("%v", err)
				}

				user.ConfirmLoggedIn()
			}
			http.Redirect(w, r, u.String(), 302)
			return
		}
	} else {
		if user.Data.LastLoggedIn.IsZero() {
			err = emails.AddUserToTabulaeTrialList(user.Data)
			if err != nil {
				// Redirect user back to login page
				log.Printf("%v", "Welcome email was not sent for "+user.Data.Email)
				log.Printf("%v", err)
			}
		}
		http.Redirect(w, r, "/api/billing/plans/trial", 302)
		return
	}

	http.Redirect(w, r, "/", 302)
	return
}
