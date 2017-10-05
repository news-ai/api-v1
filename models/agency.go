package models

import (
	"net/http"
	"time"

	"github.com/news-ai/api-v1/db"
)

type Agency struct {
	Base

	Name  string `json:"name"`
	Email string `json:"email"`
}

/*
* Public methods
 */

/*
* Create methods
 */

func (a *Agency) Create(r *http.Request, currentUser UserPostgres) (*Agency, error) {
	a.CreatedBy = currentUser.Id
	a.Created = time.Now()
	_, err := db.DB.Model(a).Returning("*").Insert()
	return a, err
}

/*
* Update methods
 */

// Function to save a new agency into App Engine
func (a *Agency) Save() (*Agency, error) {
	// Update the Updated time
	a.Updated = time.Now()
	_, err := db.DB.Model(a).Update()
	return a, err
}

/*
* Action methods
 */

func (a *Agency) FillStruct(m map[string]interface{}) error {
	for k, v := range m {
		err := SetField(a, k, v)
		if err != nil {
			return err
		}
	}
	return nil
}
