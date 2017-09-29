package models

import (
	"net/http"
	"time"
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

func (a *Agency) Create(r *http.Request, currentUser User) (*Agency, error) {
	a.CreatedBy = currentUser.Id
	a.Created = time.Now()
	_, err := a.Save()
	return a, err
}

/*
* Update methods
 */

// Function to save a new agency into App Engine
func (a *Agency) Save() (*Agency, error) {
	// Update the Updated time
	a.Updated = time.Now()
	return a, nil
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
