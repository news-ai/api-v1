package models

import (
	"net/http"
	"time"
)

type Billing struct {
	Base

	StripeId     string    `json:"-"`
	StripePlanId string    `json:"-"`
	Expires      time.Time `json:"-"`
	HasTrial     bool      `json:"-"`
	IsOnTrial    bool      `json:"-"`
	IsAgency     bool      `json:"-"`
	IsCancel     bool      `json:"-"`

	ReasonForCancel string `json:"-"`

	ReasonNotPurchase  string `json:"-"`
	FeedbackAfterTrial string `json:"-"`

	TrialEmailSent bool `json:"-"`

	CardsOnFile []string `json:"-"`
}

/*
* Public methods
 */

/*
* Create methods
 */

func (bi *Billing) Create(r *http.Request, currentUser User) (*Billing, error) {
	bi.CreatedBy = currentUser.Id
	bi.Created = time.Now()
	_, err := bi.Save()
	return bi, err
}

/*
* Update methods
 */

// Function to save a new billing into App Engine
func (bi *Billing) Save() (*Billing, error) {
	// Update the Updated time
	bi.Updated = time.Now()
	return bi, nil
}
