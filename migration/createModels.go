package main

import (
	"log"

	"github.com/news-ai/api-v1/models"
)

func createSchema() {
	for _, model := range []interface{}{&models.Agency{}, &models.Billing{}, &models.Client{}, &models.Plan{}, &models.Team{}, &models.User{}, &models.UserEmailCode{}, &models.UserInviteCode{}} {
		err := dB.CreateTable(model, nil)
		if err != nil {
			log.Printf("%v", err)
		}
	}
}
