package controllers

import (
	"log"

	"github.com/news-ai/api-v1/db"
	"github.com/news-ai/api-v1/models"
)

func CreateSchema() error {
	err := db.DB.CreateTable(&models.User{}, nil)
	if err != nil {
		log.Printf("%v", err)
	}

	return err
}

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

func GetUser(id string) (models.User, interface{}, error) {
	postgresUser := models.UserPostgres{}
	err := db.DB.Model(&postgresUser).Where("id = ?", id).Select()
	if err != nil {
		log.Printf("%v", err)
		return models.User{}, nil, err
	}

	postgresUser.Data.Id = postgresUser.Id
	return postgresUser.Data, nil, nil
}
