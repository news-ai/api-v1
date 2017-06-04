package models

import (
	"net/http"
	"time"

	"google.golang.org/appengine/log"

	"golang.org/x/net/context"

	"github.com/qedus/nds"
)

type BillingHistory struct {
	Base

	StartDate time.Time `json:"-"`
	EndDate   time.Time `json:"-"`

	Price int `json:"-"`
}

/*
* Public methods
 */

/*
* Create methods
 */

func (bh *BillingHistory) Create(c context.Context, r *http.Request, currentUser User) (*BillingHistory, error) {
	bh.CreatedBy = currentUser.Id
	bh.Created = time.Now()
	_, err := bh.Save(c)
	return bh, err
}

/*
* Update methods
 */

// Function to save a new billing into App Engine
func (bh *BillingHistory) Save(c context.Context) (*BillingHistory, error) {
	// Update the Updated time
	bh.Updated = time.Now()

	k, err := nds.Put(c, bh.BaseKey(c, "BillingHistory"), bh)
	if err != nil {
		log.Errorf(c, "%v", err)
		return nil, err
	}
	bh.Id = k.IntID()
	return bh, nil
}
