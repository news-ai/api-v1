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
	uec.CreatedBy = currentUser.Id
	uec.Created = time.Now()
	_, err := db.DB.Model(uec).Returning("*").Insert()
	return uec, err
}

/*
* Update methods
 */

func (uec *UserEmailCode) Save() (*UserEmailCode, error) {
	uec.Updated = time.Now()
	_, err := db.DB.Model(uec).Update()
	return uec, err
}

func (uec *UserEmailCode) Delete() (*UserEmailCode, error) {
	err := db.DB.Delete(uec)
	return uec, err
}
