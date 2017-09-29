package models

import (
	"time"
)

type Base struct {
	Id int64 `json:"id"`

	Type string `json:"type"`

	CreatedBy int64 `json:"createdby"`

	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
}
