package controllers

import (
	"errors"
	"log"
	"net/http"

	"github.com/news-ai/api-v1/db"
	"github.com/news-ai/api-v1/models"
)

func GetUserBilling(r *http.Request, userPostgres models.UserPostgres) (models.BillingPostgres, error) {
	if userPostgres.Data.BillingId == 0 {
		return models.BillingPostgres{}, errors.New("No billing for this user")
	}

	billingPostgres := models.BillingPostgres{}
	err := db.DB.Model(&billingPostgres).Where("id = ?", userPostgres.Data.BillingId).Select()
	if err != nil {
		log.Printf("%v", err)
		return models.BillingPostgres{}, err
	}

	billingPostgres.Data.Type = "billings"
	billingPostgres.Data.Id = billingPostgres.Id

	return billingPostgres, nil
}
