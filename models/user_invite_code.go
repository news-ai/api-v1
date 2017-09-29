package models

import (
	"net/http"
	"time"
)

type Invite struct {
	Email        string `json:"email"`
	PersonalNote string `json:"personalnote"`
}

type UserInviteCode struct {
	Base

	InviteCode string `json:"invitecode"`
	Email      string `json:"email"`
	IsUsed     bool   `json:"isused"`
}

/*
* Public methods
 */

/*
* Create methods
 */

func (uic *UserInviteCode) Create(r *http.Request, currentUser User) (*UserInviteCode, error) {
	// Create user
	uic.CreatedBy = currentUser.Id
	uic.Created = time.Now()
	uic.IsUsed = false

	_, err := uic.Save()
	return uic, err
}

/*
* Update methods
 */

// Function to save a new user into App Engine
func (uic *UserInviteCode) Save() (*UserInviteCode, error) {
	uic.Updated = time.Now()
	return uic, nil
}

// Function to save a new user into App Engine
func (uic *UserInviteCode) Delete() (*UserInviteCode, error) {
	return uic, nil
}
