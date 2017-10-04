package db

import (
	"log"
	"time"

	"github.com/go-pg/pg"
)

var DB *pg.DB

func InitDB() {
	DB = pg.Connect(&pg.Options{
		Addr:     "newsaiapitest.cnloofuhvjcp.us-east-2.rds.amazonaws.com:5432",
		User:     "newsaiapitest",
		Database: "api",
		Password: "*Q4MQNCbtlLyP",
	})

	DB.OnQueryProcessed(func(event *pg.QueryProcessedEvent) {
		query, err := event.FormattedQuery()
		if err != nil {
			panic(err)
		}

		log.Printf("%s %s", time.Since(event.StartTime), query)
	})
}
