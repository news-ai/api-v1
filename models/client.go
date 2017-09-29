package models

import (
	"net/http"
	"time"
)

type Client struct {
	Base

	Name  string   `json:"name"`
	URL   string   `json:"url"`
	Notes string   `json:"notes"`
	Tags  []string `json:"tags"`

	TeamId int64 `json:"teamid"`

	LinkedIn  string   `json:"linkedin"`
	Twitter   string   `json:"twitter"`
	Instagram string   `json:"instagram"`
	Websites  []string `json:"websites"`
	Blog      string   `json:"blog"`
}

/*
* Public methods
 */

/*
* Create methods
 */

func (cl *Client) Create(r *http.Request, currentUser User) (*Client, error) {
	cl.CreatedBy = currentUser.Id
	cl.Created = time.Now()
	_, err := cl.Save()
	return cl, err
}

/*
* Update methods
 */

// Function to save a new billing into App Engine
func (cl *Client) Save() (*Client, error) {
	// Update the Updated time
	cl.Updated = time.Now()
	return cl, nil
}
