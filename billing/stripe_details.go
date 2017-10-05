package billing

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/client"

	"github.com/news-ai/api-v1/models"
)

type Card struct {
	LastFour  string
	IsDefault bool
	Brand     stripe.CardBrand
}

type StripeError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type StripeBillingHistory struct {
	Amount  float64 `json:"amount"`
	Created string  `json:"created"`
	Paid    bool    `json:"paid"`
}

func GetCustomerBalance(userBilling *models.BillingPostgres) (int64, error) {
	sc := &client.API{}
	sc.Init(os.Getenv("STRIPE_SECRET_KEY"), nil)

	customer, err := sc.Customers.Get(userBilling.Data.StripeId, nil)
	if err != nil {
		var stripeError StripeError
		err = json.Unmarshal([]byte(err.Error()), &stripeError)
		if err != nil {
			return 0.0, errors.New("We had an error getting your user")
		}

		log.Printf("%v", err)
		return 0.0, errors.New(stripeError.Message)
	}

	return customer.Balance, nil
}

func GetCustomerBillingHistory(userBilling *models.BillingPostgres) ([]StripeBillingHistory, error) {
	sc := &client.API{}
	sc.Init(os.Getenv("STRIPE_SECRET_KEY"), nil)

	customer, err := sc.Customers.Get(userBilling.Data.StripeId, nil)
	if err != nil {
		var stripeError StripeError
		err = json.Unmarshal([]byte(err.Error()), &stripeError)
		if err != nil {
			return []StripeBillingHistory{}, errors.New("We had an error getting your user")
		}

		log.Printf("%v", err)
		return []StripeBillingHistory{}, errors.New(stripeError.Message)
	}

	params := &stripe.ChargeListParams{}
	params.Customer = customer.ID
	i := sc.Charges.List(params)

	billingHistory := []StripeBillingHistory{}

	for i.Next() {
		singleCharge := i.Charge()

		history := StripeBillingHistory{}
		history.Amount = float64(float64(singleCharge.Amount) / float64(100))
		history.Created = time.Unix(singleCharge.Created, 0).Format("2006-01-02")
		history.Paid = singleCharge.Paid

		billingHistory = append(billingHistory, history)
	}

	return billingHistory, nil
}

func GetCoupon(coupon string) (uint64, error) {
	sc := &client.API{}
	sc.Init(os.Getenv("STRIPE_SECRET_KEY"), nil)
	coupon = strings.ToUpper(coupon)

	stripeCoupon, err := sc.Coupons.Get(coupon, nil)
	if err != nil {
		var stripeError StripeError
		err = json.Unmarshal([]byte(err.Error()), &stripeError)
		if err != nil {
			return uint64(0), errors.New("Your coupon was invalid")
		}

		log.Printf("%v", err)
		return uint64(0), errors.New(stripeError.Message)
	}

	if stripeCoupon.Valid && stripeCoupon.Live {
		return stripeCoupon.Percent, nil
	}

	return uint64(0), errors.New("Your coupon was invalid or has expired")
}

func GetUserCards(userBilling *models.BillingPostgres) ([]Card, error) {
	sc := &client.API{}
	sc.Init(os.Getenv("STRIPE_SECRET_KEY"), nil)

	customer, err := sc.Customers.Get(userBilling.Data.StripeId, nil)
	if err != nil {
		var stripeError StripeError
		err = json.Unmarshal([]byte(err.Error()), &stripeError)
		if err != nil {
			return []Card{}, errors.New("We had an error getting your user")
		}

		log.Printf("%v", err)
		return []Card{}, errors.New(stripeError.Message)
	}

	cards := []Card{}
	for i := 0; i < len(customer.Sources.Values); i++ {
		newCard := Card{}
		newCard.IsDefault = customer.Sources.Values[i].Card.Default
		newCard.LastFour = customer.Sources.Values[i].Card.LastFour
		newCard.Brand = customer.Sources.Values[i].Card.Brand
		cards = append(cards, newCard)
	}

	return cards, nil
}

func AddPaymentsToCustomer(userBilling *models.BillingPostgres, stripeToken string) error {
	sc := &client.API{}
	sc.Init(os.Getenv("STRIPE_SECRET_KEY"), nil)

	params := &stripe.CustomerParams{}
	params.SetSource(stripeToken)

	_, err := sc.Customers.Update(
		userBilling.Data.StripeId,
		params,
	)

	if err != nil {
		var stripeError StripeError
		err = json.Unmarshal([]byte(err.Error()), &stripeError)
		if err != nil {
			return errors.New("We had an error getting your user")
		}

		log.Printf("%v", err)
		return errors.New(stripeError.Message)
	}

	newCustomer, err := sc.Customers.Get(userBilling.Data.StripeId, nil)
	if err != nil {
		log.Printf("%v", err)
		return err
	}

	userBilling.Data.CardsOnFile = []string{}
	for i := 0; i < len(newCustomer.Sources.Values); i++ {
		userBilling.Data.CardsOnFile = append(userBilling.Data.CardsOnFile, newCustomer.Sources.Values[i].ID)
	}
	userBilling.Save()

	return nil
}
