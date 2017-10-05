package models

import (
	"net/http"
	"time"

	"github.com/news-ai/api-v1/db"
)

type Team struct {
	Base

	Name string `json:"name"`

	AgencyId int64 `json:"agencyid" apiModel:"Agency"`

	MaxMembers int `json:"maxmembers" apiModel:"User"`

	Members []int64 `json:"members" apiModel:"User"`
	Admins  []int64 `json:"admins" apiModel:"User"`
}

/*
* Public methods
 */

/*
* Create methods
 */

// Function to create a new team into App Engine
func (t *Team) Create(r *http.Request, currentUser UserPostgres) (*Team, error) {
	t.CreatedBy = currentUser.Id
	t.Created = time.Now()
	_, err := db.DB.Model(t).Returning("*").Insert()
	return t, err
}

/*
* Update methods
 */

// Function to save a new team into App Engine
func (t *Team) Save() (*Team, error) {
	// Update the Updated time
	t.Updated = time.Now()
	_, err := db.DB.Model(t).Update()
	return t, err
}
