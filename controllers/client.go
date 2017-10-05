package controllers

import (
	"errors"
	"log"
	"net/http"

	"github.com/news-ai/api-v1/db"
	"github.com/news-ai/api-v1/models"

	"github.com/news-ai/web/utilities"
)

/*
* Private methods
 */

/*
* Get methods
 */

func getClient(id int64) (models.Client, error) {
	if id == 0 {
		return models.Client{}, errors.New("datastore: no such entity")
	}

	client := models.Client{}
	err := db.DB.Model(&client).Where("id = ?", id).Select()
	if err != nil {
		log.Printf("%v", err)
		return models.Client{}, err
	}

	if !client.Created.IsZero() {
		client.Type = "clients"
		return client, nil
	}

	return models.Client{}, errors.New("No client by this id")
}

/*
* Public methods
 */

/*
* Get methods
 */

func GetClients(r *http.Request) ([]models.Client, interface{}, int, int, error) {
	// Now if user is not querying then check
	user, err := GetCurrentUser(r)
	if err != nil {
		log.Printf("%v", err)
		return []models.Client{}, nil, 0, 0, err
	}

	if !user.Data.IsAdmin {
		return []models.Client{}, nil, 0, 0, errors.New("Forbidden")
	}

	clients := []models.Client{}
	err = db.DB.Model(&clients).Where("created_by = ?", user.Id).Select()
	if err != nil {
		log.Printf("%v", err)
		return []models.Client{}, nil, 0, 0, err
	}

	for i := 0; i < len(clients); i++ {
		clients[i].Type = "clients"
	}

	return clients, nil, len(clients), 0, nil
}

func GetClient(id string) (models.Client, interface{}, error) {
	// Get the details of a client
	currentId, err := utilities.StringIdToInt(id)
	if err != nil {
		log.Printf("%v", err)
		return models.Client{}, nil, err
	}

	client, err := getClient(currentId)
	if err != nil {
		log.Printf("%v", err)
		return models.Client{}, nil, err
	}

	return client, nil, nil
}
