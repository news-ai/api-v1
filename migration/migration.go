package main

import (
	"github.com/go-pg/pg"
)

var dB *pg.DB

func initDB() {
	dB = pg.Connect(&pg.Options{
		Addr:     "newsaiapitest.cnloofuhvjcp.us-east-2.rds.amazonaws.com:5432",
		User:     "newsaiapitest",
		Database: "api",
		Password: "*Q4MQNCbtlLyP",
	})
}

func main() {
	initDB()
	// getDatastoreAndInsertIntoPostgres()
	createSchema()
}
