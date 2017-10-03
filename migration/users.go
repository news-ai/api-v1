package main

import (
	"fmt"
	"log"

	"golang.org/x/net/context"

	"cloud.google.com/go/datastore"

	"github.com/news-ai/api-v1/models"
)

func getDatastoreResource() ([]*models.User, []*datastore.Key, error) {
	ctx := context.Background()
	client, err := datastore.NewClient(ctx, "newsai-1166")
	if err != nil {
		log.Printf("%v", err)
	}

	var users []*models.User
	keys, err := client.GetAll(ctx, datastore.NewQuery("User"), &users)
	return users, keys, err
}

func insertIntoPostgres(user *models.UserPostgres) error {
	initDB()
	_, err := dB.Model(user).Returning("*").Insert()
	return err
}

func getDatastoreAndInsertIntoPostgres() {
	users, datastoreKeys, err := getDatastoreResource()
	if err != nil {
		log.Printf("%v", err)
		return
	}

	for i, key := range datastoreKeys {
		fmt.Println(key)
		fmt.Println(users[i])
		users[i].Id = key.ID
		userPostgres := models.UserPostgres{}
		userPostgres.Data = *users[i]
		err = insertIntoPostgres(&userPostgres)
		if err != nil {
			log.Printf("%v", err)
			return
		}
	}
}
