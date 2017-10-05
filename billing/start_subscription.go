package billing

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/client"

	"github.com/news-ai/api-v1/models"
	"github.com/news-ai/tabulae-v1/emails"
)

func AddFreeTrialToUser(r *http.Request, user models.UserPostgres, plan string) (int64, error) {
	sc := &client.API{}
	sc.Init(os.Getenv("STRIPE_SECRET_KEY"), nil)

	// https://stripe.com/docs/api
	// Create new customer in Stripe
	params := &stripe.CustomerParams{
		Email:    user.Data.Email,
		Plan:     plan + "-trial",
		Quantity: uint64(1),
	}

	customer, err := sc.Customers.New(params)
	if err != nil {
		log.Printf("%v", err)
		return 0, err
	}

	_, billingId, err := user.SetStripeId(user, customer.ID, plan, true, true)
	if err != nil {
		log.Printf("%v", err)
		return billingId, err
	}
	return billingId, nil
}

func AddPlanToUser(r *http.Request, user models.UserPostgres, userBilling *models.BillingPostgres, plan string, duration string, coupon string, originalPlan string) error {
	sc := &client.API{}
	sc.Init(os.Getenv("STRIPE_SECRET_KEY"), nil)

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

	// Only considers plans currently that moving from trial. Not changing plans.
	// Cancel all past subscriptions they had
	for i := 0; i < len(customer.Subs.Values); i++ {
		sc.Subs.Cancel(customer.Subs.Values[i].ID, nil)
	}

	// Start a new subscription without trial (they already went through the trial)
	params := &stripe.SubParams{
		Customer: customer.ID,
		Plan:     plan,
	}

	if duration == "annually" {
		params.Plan = plan + "-yearly"
	}

	if coupon != "" {
		coupon = strings.ToUpper(coupon)
		params.Coupon = coupon
	}

	if strings.ToLower(coupon) == "favorites" && duration == "annually" {
		return errors.New("Sorry - you can't use this coupon code on a yearly plan. Please switch the monthly one to use this!")
	}

	if strings.ToLower(coupon) == "prcouture" && duration == "annually" {
		return errors.New("Sorry - you can't use this coupon code on a yearly plan. Please switch the monthly one to use this!")
	}

	if strings.ToLower(coupon) == "curious" && duration == "annually" {
		return errors.New("Sorry - you can't use this coupon code on a yearly plan. Please switch the monthly one to use this!")
	}

	if strings.ToLower(coupon) == "prconsultants" && duration == "annually" {
		return errors.New("Sorry - you can't use this coupon code on a yearly plan. Please switch the monthly one to use this!")
	}

	newSub, err := sc.Subs.New(params)
	if err != nil {
		var stripeError StripeError
		err = json.Unmarshal([]byte(err.Error()), &stripeError)
		if err != nil {
			log.Printf("%v", err)
			return errors.New("We had an error setting your subscription")
		}

		log.Printf("%v", err)
		return errors.New(stripeError.Message)
	}

	// Return if there are any errors
	expiresAt := time.Unix(newSub.PeriodEnd, 0)
	userBilling.Data.Expires = expiresAt
	userBilling.Data.StripePlanId = plan
	userBilling.Data.IsOnTrial = false
	userBilling.Save()

	// Set the user to be an active being on the platform again
	user.Data.IsActive = true
	user.Save()

	currentPrice := PlanAndDurationToPrice(originalPlan, duration)
	billAmount := "$" + fmt.Sprintf("%0.2f", currentPrice)
	paidAmount := "$" + fmt.Sprintf("%0.2f", currentPrice)

	ExpiresAt := expiresAt.Format("2006-01-02")

	emailDuration := "a monthly"
	if duration == "annually" {
		emailDuration = "an annual"
	}

	// Email confirmation
	err = emails.AddUserToTabulaePremiumList(user.Data, originalPlan, emailDuration, ExpiresAt, billAmount, paidAmount)
	if err != nil {
		log.Printf("%v", err)
	}

	return nil
}
