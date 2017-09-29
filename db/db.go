package db

import (
	"github.com/go-pg/pg"
)

var DB *pg.DB

func InitDB() {
	DB = pg.Connect(&pg.Options{
		User:     "abhiagarwal",
		Database: "abhiagarwal",
	})
}
