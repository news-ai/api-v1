package controllers

import (
	"errors"
	"net/http"

	"golang.org/x/net/context"

	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"

	"github.com/qedus/nds"

	"github.com/news-ai/api/models"

	"github.com/news-ai/web/utilities"
)

/*
* Private methods
 */

/*
* Get methods
 */

func getBillingHistory(c context.Context, id int64) (models.BillingHistory, error) {
	if id == 0 {
		return models.BillingHistory{}, errors.New("datastore: no such entity")
	}
	// Get the publication details by id
	var billingHistory models.BillingHistory
	billingHistoryId := datastore.NewKey(c, "BillingHistory", "", id, nil)

	err := nds.Get(c, billingHistoryId, &billingHistory)

	if err != nil {
		log.Errorf(c, "%v", err)
		return models.BillingHistory{}, err
	}

	if !billingHistory.Created.IsZero() {
		billingHistory.Format(billingHistoryId, "billinghistories")
		return billingHistory, nil
	}
	return models.BillingHistory{}, errors.New("No billing histoy by this id")
}

/*
* Public methods
 */

/*
* Get methods
 */

func GetBillingHistory(c context.Context, r *http.Request, id string) (models.BillingHistory, interface{}, error) {
	// Get the details of the current user
	currentId, err := utilities.StringIdToInt(id)
	if err != nil {
		log.Errorf(c, "%v", err)
		return models.BillingHistory{}, nil, err
	}

	billingHistory, err := getBillingHistory(c, currentId)
	if err != nil {
		log.Errorf(c, "%v", err)
		return models.BillingHistory{}, nil, err
	}

	return billingHistory, nil, nil
}

func GetUserBillingHistory(c context.Context, r *http.Request) ([]models.BillingHistory, interface{}, int, error) {
	user, err := GetCurrentUser(c, r)
	if err != nil {
		log.Errorf(c, "%v", err)
		return []models.BillingHistory{}, nil, 0, err
	}

	query := datastore.NewQuery("BillingHistory").Filter("CreatedBy =", user.Id)
	query = constructQuery(query, r)
	ks, err := query.KeysOnly().GetAll(c, nil)
	if err != nil {
		log.Errorf(c, "%v", err)
		return []models.BillingHistory{}, nil, 0, err
	}

	var userBillingHistories []models.BillingHistory
	userBillingHistories = make([]models.BillingHistory, len(ks))

	err = nds.GetMulti(c, ks, userBillingHistories)
	if err != nil {
		log.Infof(c, "%v", err)
		return []models.BillingHistory{}, nil, 0, err
	}

	for i := 0; i < len(userBillingHistories); i++ {
		userBillingHistories[i].Format(ks[i], "billinghistories")
	}

	return userBillingHistories, nil, len(userBillingHistories), nil
}
