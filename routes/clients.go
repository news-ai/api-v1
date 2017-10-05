package routes

import (
	"errors"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/pquerna/ffjson/ffjson"

	"github.com/news-ai/api-v1/controllers"

	"github.com/news-ai/web/api"
	nError "github.com/news-ai/web/errors"
)

func handleClient(r *http.Request, id string) (interface{}, error) {
	switch r.Method {
	case "GET":
		return api.BaseSingleResponseHandler(controllers.GetClient(id))
	}
	return nil, errors.New("method not implemented")
}

func handleClients(r *http.Request) (interface{}, error) {
	switch r.Method {
	case "GET":
		val, included, count, total, err := controllers.GetClients(r)
		return api.BaseResponseHandler(val, included, count, total, err, r)
	}
	return nil, errors.New("method not implemented")
}

// Handler for when the user wants all the agencies.
func ClientsHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")
	val, err := handleClients(r)

	if err == nil {
		err = ffjson.NewEncoder(w).Encode(val)
	}

	if err != nil {
		nError.ReturnError(w, http.StatusInternalServerError, "Client handling error", err.Error())
	}
	return
}

// Handler for when there is a key present after /users/<id> route.
func ClientHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")
	id := ps.ByName("id")
	val, err := handleClient(r, id)

	if err == nil {
		err = ffjson.NewEncoder(w).Encode(val)
	}

	if err != nil {
		nError.ReturnError(w, http.StatusInternalServerError, "Client handling error", err.Error())
	}
	return
}
