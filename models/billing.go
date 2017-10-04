package models

import (
	"net/http"
	"time"

	"github.com/news-ai/api-v1/db"
)

type Billing struct {
	Base

	StripeId     string    `json:"stripeid"`
	StripePlanId string    `json:"stripeplanid"`
	Expires      time.Time `json:"expires"`
	HasTrial     bool      `json:"hastrial"`
	IsOnTrial    bool      `json:"isontrial"`
	IsAgency     bool      `json:"isagency"`
	IsCancel     bool      `json:"iscancel"`

	ReasonForCancel string `json:"reasonforcancel"`

	ReasonNotPurchase  string `json:"reasonnotpurchase"`
	FeedbackAfterTrial string `json:"feedbackaftertrial"`

	TrialEmailSent bool `json:"-"`

	CardsOnFile []string `json:"cardsonfile"`
}

type BillingPostgres struct {
	Id int64

	Data Billing
}

/*
* Public methods
 */

/*
* Create methods
 */

func (bi *BillingPostgres) Create(r *http.Request, currentUser User) (*BillingPostgres, error) {
	bi.Data.CreatedBy = currentUser.Id
	bi.Data.Created = time.Now()
	_, err := db.DB.Model(bi).Returning("*").Insert()
	return bi, err
}

/*
* Update methods
 */

// Function to save a new billing into App Engine
func (bi *BillingPostgres) Save() (*BillingPostgres, error) {
	// Update the Updated time
	bi.Data.Updated = time.Now()
	_, err := db.DB.Model(bi).Set("data = ?data").Where("id = ?id").Returning("*").Update()
	return bi, err
}
