package controllers

import (
	"errors"
	"log"
	"net/http"

	gcontext "github.com/gorilla/context"

	"github.com/news-ai/api-v1/db"
	"github.com/news-ai/api-v1/models"
)

func GetUsers() ([]models.User, interface{}, int, int, error) {
	postgresUsers := []models.UserPostgres{}
	err := db.DB.Model(&postgresUsers).Select()
	if err != nil {
		log.Printf("%v", err)
	}

	users := []models.User{}
	for i := 0; i < len(postgresUsers); i++ {
		postgresUsers[i].Data.Id = postgresUsers[i].Id
		users = append(users, postgresUsers[i].Data)
	}

	return users, nil, 0, 0, nil
}

func GetUserByEmail(email string) (models.UserPostgres, error) {
	postgresUser := models.UserPostgres{}
	err := db.DB.Model(&postgresUser).Where("data->>'email' = '" + email + "'").Select()
	if err != nil {
		log.Printf("%v", err)
		return models.UserPostgres{}, err
	}

	postgresUser.Data.Id = postgresUser.Id
	return postgresUser, nil
}

func GetUser(r *http.Request, id string) (models.User, interface{}, error) {
	switch id {
	case "me":
		user, err := GetCurrentUser(r)
		if err != nil {
			log.Printf("%v", err)
			return models.User{}, nil, err
		}
		return user.Data, nil, err
	default:
		postgresUser := models.UserPostgres{}
		err := db.DB.Model(&postgresUser).Where("id = ?", id).Select()
		if err != nil {
			log.Printf("%v", err)
			return models.User{}, nil, err
		}

		postgresUser.Data.Id = postgresUser.Id
		return postgresUser.Data, nil, nil
	}
}

func Update(r *http.Request, u *models.UserPostgres) (*models.UserPostgres, error) {
	return u, nil
}

func AddUserToContext(r *http.Request, email string) {
	_, ok := gcontext.GetOk(r, "user")
	if !ok {
		user, _ := GetUserByEmail(email)
		gcontext.Set(r, "user", user)
		Update(r, &user)
	} else {
		user := gcontext.Get(r, "user").(models.UserPostgres)
		Update(r, &user)
	}
}

func GetCurrentUser(r *http.Request) (models.UserPostgres, error) {
	// Get the current user
	_, ok := gcontext.GetOk(r, "user")
	if !ok {
		return models.UserPostgres{}, errors.New("No user logged in")
	}
	user := gcontext.Get(r, "user").(models.UserPostgres)
	return user, nil
}
