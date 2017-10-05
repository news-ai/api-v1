package billing

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"time"

	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/client"

	"github.com/news-ai/api-v1/models"
)

func SwitchUserPlanPreview(userBilling *models.BillingPostgres, duration, newPlan string) (int64, error) {
	sc := &client.API{}
	sc.Init(os.Getenv("STRIPE_SECRET_KEY"), nil)

	customer, err := sc.Customers.Get(userBilling.Data.StripeId, nil)
	if err != nil {
		var stripeError StripeError
		err = json.Unmarshal([]byte(err.Error()), &stripeError)
		if err != nil {
			log.Printf("%v", err)
			return 0.0, errors.New("We had an error getting your user")
		}

		log.Printf("%v", err)
		return 0.0, errors.New(stripeError.Message)
	}

	if duration == "annually" {
		newPlan = newPlan + "-yearly"
	}

	if customer.Subs.Count > 0 {
		prorationDate := time.Now().Unix()

		invoiceParams := &stripe.InvoiceParams{
			Customer:         customer.ID,
			Sub:              customer.Subs.Values[0].ID,
			SubPlan:          newPlan,
			SubProrationDate: prorationDate,
		}
		invoice, err := sc.Invoices.GetNext(invoiceParams)

		if err != nil {
			log.Printf("%v", err)
			return 0.0, err
		}

		var cost int64 = 0
		for _, invoiceItem := range invoice.Lines.Values {
			if invoiceItem.Period.Start == prorationDate {
				cost += invoiceItem.Amount
			}
		}

		return cost, nil
	}

	return 0.00, nil
}

func SwitchUserPlan(userBilling *models.BillingPostgres, newPlan string) error {
	sc := &client.API{}
	sc.Init(os.Getenv("STRIPE_SECRET_KEY"), nil)

	_, err := sc.Customers.Get(userBilling.Data.StripeId, nil)
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

	return nil
}
