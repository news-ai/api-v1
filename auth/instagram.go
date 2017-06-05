package auth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"

	"github.com/news-ai/oauth2/instagram"
	"golang.org/x/oauth2"

	apiControllers "github.com/news-ai/api/controllers"

	"github.com/news-ai/web/utilities"

	"github.com/julienschmidt/httprouter"
)

var (
	instagramOauthConfig = &oauth2.Config{
		RedirectURL:  "https://tabulae.newsai.org/api/internal_auth/instagramcallback",
		ClientID:     os.Getenv("INSTAGRAMAUTHKEY"),
		ClientSecret: os.Getenv("INSTAGRAMAUTHSECRET"),
		Endpoint:     instagram.Endpoint,
	}
)

func InstagramLoginHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	c := appengine.NewContext(r)

	// Make sure the user has been logged in when at instagram auth
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

	session.Values["instagram_state"] = state

	if r.URL.Query().Get("next") != "" {
		session.Values["next"] = r.URL.Query().Get("next")
	}

	err = session.Save(r, w)
	if err != nil {
		log.Errorf(c, "%v", err)
	}

	// Scope
	instagramScope := oauth2.SetAuthURLParam("scope", "public_content relationships")

	// Redirect the user to the login page
	url := instagramOauthConfig.AuthCodeURL(state, instagramScope)
	http.Redirect(w, r, url, 302)
	return
}

func InstagramCallbackHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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

	if r.URL.Query().Get("state") != session.Values["instagram_state"] {
		fmt.Fprintln(w, "no state match; possible csrf OR cookies not enabled")
		return
	}

	tkn, err := instagramOauthConfig.Exchange(c, r.URL.Query().Get("code"))

	if err != nil {
		fmt.Fprintln(w, "there was an issue getting your token")
		return
	}

	if !tkn.Valid() {
		fmt.Fprintln(w, "retreived invalid token")
		return
	}

	client := instagramOauthConfig.Client(c, tkn)
	req, err := http.NewRequest("GET", "https://api.instagram.com/v1/users/self/?access_token="+tkn.AccessToken, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
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

	var instagramUser struct {
		Data struct {
			ID             string `json:"id"`
			Username       string `json:"username"`
			FullName       string `json:"full_name"`
			ProfilePicture string `json:"profile_picture"`
			Bio            string `json:"bio"`
			Website        string `json:"website"`
			Counts         struct {
				Media      int `json:"media"`
				Follows    int `json:"follows"`
				FollowedBy int `json:"followed_by"`
			} `json:"counts"`
		} `json:"data"`
	}

	err = json.Unmarshal(str, &instagramUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	currentUser.InstagramId = instagramUser.Data.ID
	currentUser.InstagramAuthKey = tkn.AccessToken
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
