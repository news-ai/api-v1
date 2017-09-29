package models

import (
	"net/http"
	"time"
)

type UserEmail struct {
	Email string `json:"email"`
}

type UserEmailCode struct {
	Base

	InviteCode string `json:"invitecode"`
	Email      string `json:"email"`
}

/*
* Public methods
 */

/*
* Create methods
 */

func (uec *UserEmailCode) Create(r *http.Request, currentUser User) (*UserEmailCode, error) {
	// Create user
	uec.CreatedBy = currentUser.Id
	uec.Created = time.Now()

	_, err := uec.Save()
	return uec, err
}

/*
* Update methods
 */

// Function to save a new user into App Engine
func (uec *UserEmailCode) Save() (*UserEmailCode, error) {
	uec.Updated = time.Now()
	return uec, nil
}

// Function to save a new user into App Engine
func (uec *UserEmailCode) Delete() (*UserEmailCode, error) {
	return uec, nil
}
