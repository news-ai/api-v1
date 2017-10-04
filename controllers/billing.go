package controllers

import (
	"errors"
	"log"
	"net/http"

	"github.com/news-ai/api-v1/models"
)

func GetUserBilling(r *http.Request, userPostgres models.UserPostgres) (models.BillingPostgres, error) {
	if userPostgres.Data.BillingId == 0 {
		return models.Billing{}, errors.New("No billing for this user")
	}
	// Get the billing by id
	var billing models.Billing
	billingId := datastore.NewKey(c, "Billing", "", userPostgres.Data.BillingId, nil)
	err := nds.Get(c, billingId, &billing)
	if err != nil {
		log.Printf("%v", err)
		return models.Billing{}, err
	}

	billing.Format(billingId, "billings")

	return billing, nil
}
