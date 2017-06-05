package auth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/linkedin"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"

	apiControllers "github.com/news-ai/api/controllers"

	"github.com/news-ai/web/utilities"

	"github.com/julienschmidt/httprouter"
)

var (
	linkedinOauthConfig = &oauth2.Config{
		RedirectURL:  "https://tabulae.newsai.org/api/internal_auth/linkedincallback",
		ClientID:     os.Getenv("LINKEDINAUTHKEY"),
		ClientSecret: os.Getenv("LINKEDINAUTHSECRET"),
		Endpoint:     linkedin.Endpoint,
	}
)

func LinkedinLoginHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	c := appengine.NewContext(r)

	// Make sure the user has been logged in when at linkedin auth
	_, err := apiControllers.GetCurrentUser(c, r)
	if err != nil {
		log.Errorf(c, "%v", err)
		fmt.Fprintln(w, "user not logged in")
		return
	}

	// Generate a random state that we identify the user with
	state := utilities.RandToken()

	// Save the session for each of the users
	session, err := Store.Get(r, "sess")
	if err != nil {
		log.Errorf(c, "%v", err)
	}

	session.Values["linkedin_state"] = state

	if r.URL.Query().Get("next") != "" {
		session.Values["next"] = r.URL.Query().Get("next")
	}

	err = session.Save(r, w)
	if err != nil {
		log.Errorf(c, "%v", err)
	}

	// Redirect the user to the login page
	url := linkedinOauthConfig.AuthCodeURL(state)
	http.Redirect(w, r, url, 302)
	return
}

func LinkedinCallbackHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	c := appengine.NewContext(r)
	currentUser, err := apiControllers.GetCurrentUser(c, r)
	if err != nil {
		log.Infof(c, "%v", err)
		fmt.Fprintln(w, "user not logged in")
		return
	}

	session, err := Store.Get(r, "sess")
	if err != nil {
		log.Infof(c, "%v", err)
		fmt.Fprintln(w, "aborted")
		return
	}

	if r.URL.Query().Get("state") != session.Values["linkedin_state"] {
		fmt.Fprintln(w, "no state match; possible csrf OR cookies not enabled")
		return
	}

	tkn, err := linkedinOauthConfig.Exchange(c, r.URL.Query().Get("code"))
	if err != nil {
		fmt.Fprintln(w, "there was an issue getting your token")
		return
	}

	if !tkn.Valid() {
		fmt.Fprintln(w, "retreived invalid token")
		return
	}

	client := linkedinOauthConfig.Client(c, tkn)
	req, err := http.NewRequest("GET", "https://api.linkedin.com/v1/people/~:(email-address,first-name,last-name,id,headline)?format=json", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	req.Header.Set("Bearer", tkn.AccessToken)
	response, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	defer response.Body.Close()
	str, err := ioutil.ReadAll(response.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var linkedinUser struct {
		Id          string
		FirstName   string `json:"firstName"`
		LastName    string `json:"lastName"`
		Headline    string
		Email       string `json:"emailAddress"`
		AccessToken string
	}

	err = json.Unmarshal(str, &linkedinUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	linkedinUser.AccessToken = tkn.AccessToken
	currentUser.LinkedinId = linkedinUser.Id
	currentUser.LinkedinAuthKey = tkn.AccessToken
	currentUser.Save(c)

	if session.Values["next"] != nil {
		returnURL := session.Values["next"].(string)
		u, err := url.Parse(returnURL)
		if err != nil {
			http.Redirect(w, r, returnURL, 302)
		}
		http.Redirect(w, r, u.String(), 302)
		return
	}

	http.Redirect(w, r, "/", 302)
	return
}
