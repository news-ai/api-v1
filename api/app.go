package main

import (
	"net/http"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/context"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/cors"

	"github.com/news-ai/web/api"
)

func main() {
	// Setting up Negroni Router
	app := negroni.New()
	app.Use(negroni.NewRecovery())
	app.Use(negroni.NewLogger())

	// CORs
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"https://newsai.org", "https://newsai.co", "http://localhost:3000", "https://site.newsai.org", "http://site.newsai.org", "https://site.newsai.co", "http://site.newsai.co", "http://tabulae.newsai.co", "https://tabulae.newsai.co", "http://tabulae-dev.newsai.co", "https://tabulae-dev.newsai.co", "https://internal.newsai.org"},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"},
		Debug:            true,
		AllowedHeaders:   []string{"*"},
	})
	app.Use(c)

	// Initialize router
	router := httprouter.New()
	router.NotFound = http.HandlerFunc(api.NotFound)

	// Not found Handler
	router.GET("/", api.NotFoundHandler)
	router.GET("/api", api.NotFoundHandler)

	app.UseHandler(router)

	// Register the app router
	http.Handle("/", context.ClearHandler(app))
	http.ListenAndServe(":8080", nil)
}
