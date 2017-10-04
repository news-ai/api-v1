package billing

import (
	"encoding/json"
	"errors"
	"log"
	"os"

	"github.com/stripe/stripe-go/client"

	"github.com/news-ai/api-v1/models"
)

func CancelPlanOfUser(userBilling *models.BillingPostgres) error {
	sc := &client.API{}
	sc.Init(os.Getenv("STRIPE_SECRET_KEY"), nil)

	if userBilling.Data.IsOnTrial {
		return errors.New("Can not cancel a trial")
	}

	customer, err := sc.Customers.Get(userBilling.Data.StripeId, nil)
	if err != nil {
		var stripeError StripeError
		err = json.Unmarshal([]byte(err.Error()), &stripeError)
		if err != nil {
			log.Printf("%v", err)
			return errors.New("We had an error getting your user")
		}

		log.Printf("%v", err)
		return errors.New(stripeError.Message)
	}

	// Cancel all plans they might have (they should only have one)
	for i := 0; i < len(customer.Subs.Values); i++ {
		sc.Subs.Cancel(customer.Subs.Values[i].ID, nil)
	}

	userBilling.Data.IsCancel = true
	userBilling.Save()

	// Send an email to the user saying that the package will be canceled. Their account will be inactive on
	// their "Expires" date.

	return nil
}
