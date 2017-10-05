package models

import (
	"net/http"
	"time"

	"github.com/news-ai/api-v1/db"
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

func (uic *UserInviteCode) Create(r *http.Request, currentUser UserPostgres) (*UserInviteCode, error) {
	// Create user
	uic.CreatedBy = currentUser.Id
	uic.Created = time.Now()
	uic.IsUsed = false

	_, err := db.DB.Model(uic).Returning("*").Insert()
	return uic, err
}

/*
* Update methods
 */

// Function to save a new user into App Engine
func (uic *UserInviteCode) Save() (*UserInviteCode, error) {
	uic.Updated = time.Now()
	_, err := db.DB.Model(uic).Update()
	return uic, err
}

// Function to save a new user into App Engine
func (uic *UserInviteCode) Delete() (*UserInviteCode, error) {
	return uic, nil
}
