package db

import (
	"github.com/go-pg/pg"
)

var DB *pg.DB

func InitDB() {
	DB = pg.Connect(&pg.Options{
		Addr:     "newsaiapitest.cnloofuhvjcp.us-east-2.rds.amazonaws.com:5432",
		User:     "newsaiapitest",
		Database: "newsaiapitest",
		Password: "*Q4MQNCbtlLyP",
	})
}
