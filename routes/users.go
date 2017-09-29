package routes

import (
	"errors"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/pquerna/ffjson/ffjson"

	"github.com/news-ai/api-v1/controllers"

	"github.com/news-ai/web/api"
	nError "github.com/news-ai/web/errors"
)

func handleUsers(r *http.Request) (interface{}, error) {
	switch r.Method {
	case "GET":
		val, included, count, total, err := controllers.GetUsers()
		return api.BaseResponseHandler(val, included, count, total, err, r)
	}
	return nil, errors.New("method not implemented")
}

// Handler for when the user wants all the users.
func UsersHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")
	val, err := handleUsers(r)

	log.Printf("error: %v", err)

	if err == nil {
		err = ffjson.NewEncoder(w).Encode(val)
	}

	if err != nil {
		nError.ReturnError(w, http.StatusInternalServerError, "User handling error", err.Error())
	}
	return
}
