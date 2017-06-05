package billing

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"

	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/client"

	"github.com/news-ai/api/models"
	"github.com/news-ai/tabulae/emails"
)

func AddFreeTrialToUser(r *http.Request, user models.User, plan string) (int64, error) {
	c := appengine.NewContext(r)
	httpClient := urlfetch.Client(c)
	sc := client.New(os.Getenv("STRIPE_SECRET_KEY"), stripe.NewBackends(httpClient))

	// https://stripe.com/docs/api
	// Create new customer in Stripe
	params := &stripe.CustomerParams{
		Email:    user.Email,
		Plan:     plan + "-trial",
		Quantity: uint64(1),
	}

	customer, err := sc.Customers.New(params)
	if err != nil {
		log.Errorf(c, "%v", err)
		return 0, err
	}

	_, billingId, err := user.SetStripeId(c, r, user, customer.ID, plan, true, true)
	if err != nil {
		log.Errorf(c, "%v", err)
		return billingId, err
	}
	return billingId, nil
}

func CancelPlanOfUser(r *http.Request, user models.User, userBilling *models.Billing) error {
	c := appengine.NewContext(r)
	httpClient := urlfetch.Client(c)
	sc := client.New(os.Getenv("STRIPE_SECRET_KEY"), stripe.NewBackends(httpClient))

	if userBilling.IsOnTrial {
		return errors.New("Can not cancel a trial")
	}

	customer, err := sc.Customers.Get(userBilling.StripeId, nil)
	if err != nil {
		var stripeError StripeError
		err = json.Unmarshal([]byte(err.Error()), &stripeError)
		if err != nil {
			log.Errorf(c, "%v", err)
			return errors.New("We had an error getting your user")
		}

		log.Errorf(c, "%v", err)
		return errors.New(stripeError.Message)
	}

	// Cancel all plans they might have (they should only have one)
	for i := 0; i < len(customer.Subs.Values); i++ {
		sc.Subs.Cancel(customer.Subs.Values[i].ID, nil)
	}

	userBilling.IsCancel = true
	userBilling.Save(c)

	// Send an email to the user saying that the package will be canceled. Their account will be inactive on
	// their "Expires" date.

	return nil
}

func AddPlanToUser(r *http.Request, user models.User, userBilling *models.Billing, plan string, duration string, coupon string, originalPlan string) error {
	c := appengine.NewContext(r)
	httpClient := urlfetch.Client(c)
	sc := client.New(os.Getenv("STRIPE_SECRET_KEY"), stripe.NewBackends(httpClient))

	customer, err := sc.Customers.Get(userBilling.StripeId, nil)
	if err != nil {
		var stripeError StripeError
		err = json.Unmarshal([]byte(err.Error()), &stripeError)
		if err != nil {
			log.Errorf(c, "%v", err)
			return errors.New("We had an error getting your user")
		}

		log.Errorf(c, "%v", err)
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
			log.Errorf(c, "%v", err)
			return errors.New("We had an error setting your subscription")
		}

		log.Errorf(c, "%v", err)
		return errors.New(stripeError.Message)
	}

	// Return if there are any errors
	expiresAt := time.Unix(newSub.PeriodEnd, 0)
	userBilling.Expires = expiresAt
	userBilling.StripePlanId = plan
	userBilling.IsOnTrial = false
	userBilling.Save(c)

	// Set the user to be an active being on the platform again
	user.IsActive = true
	user.Save(c)

	currentPrice := PlanAndDurationToPrice(originalPlan, duration)
	billAmount := "$" + fmt.Sprintf("%0.2f", currentPrice)
	paidAmount := "$" + fmt.Sprintf("%0.2f", currentPrice)

	ExpiresAt := expiresAt.Format("2006-01-02")

	emailDuration := "a monthly"
	if duration == "annually" {
		emailDuration = "an annual"
	}

	// Email confirmation
	err = emails.AddUserToTabulaePremiumList(c, user, originalPlan, emailDuration, ExpiresAt, billAmount, paidAmount)
	if err != nil {
		log.Errorf(c, "%v", err)
	}

	return nil
}

func SwitchUserPlanPreview(r *http.Request, user models.User, userBilling *models.Billing, duration, newPlan string) (int64, error) {
	c := appengine.NewContext(r)
	httpClient := urlfetch.Client(c)
	sc := client.New(os.Getenv("STRIPE_SECRET_KEY"), stripe.NewBackends(httpClient))

	customer, err := sc.Customers.Get(userBilling.StripeId, nil)
	if err != nil {
		var stripeError StripeError
		err = json.Unmarshal([]byte(err.Error()), &stripeError)
		if err != nil {
			log.Errorf(c, "%v", err)
			return 0.0, errors.New("We had an error getting your user")
		}

		log.Errorf(c, "%v", err)
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
			log.Errorf(c, "%v", err)
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

func SwitchUserPlan(r *http.Request, user models.User, userBilling *models.Billing, newPlan string) error {
	c := appengine.NewContext(r)
	httpClient := urlfetch.Client(c)
	sc := client.New(os.Getenv("STRIPE_SECRET_KEY"), stripe.NewBackends(httpClient))

	_, err := sc.Customers.Get(userBilling.StripeId, nil)
	if err != nil {
		var stripeError StripeError
		err = json.Unmarshal([]byte(err.Error()), &stripeError)
		if err != nil {
			log.Errorf(c, "%v", err)
			return errors.New("We had an error getting your user")
		}

		log.Errorf(c, "%v", err)
		return errors.New(stripeError.Message)
	}

	return nil
}
