package models

import (
	"net/http"
	"time"
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
