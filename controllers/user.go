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
	users := []models.User{}
	err := db.DB.Model(&users).Select()
	if err != nil {
		log.Printf("%v", err)
	}

	return users, nil, 0, 0, err
}
