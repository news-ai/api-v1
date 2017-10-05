package main

import (
	"log"
	"net/http"
	"os"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/context"
	"github.com/gorilla/csrf"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/cors"
	"github.com/unrolled/secure"

	"github.com/news-ai/api-v1/auth"
	"github.com/news-ai/api-v1/db"
	"github.com/news-ai/api-v1/middleware"
	"github.com/news-ai/api-v1/routes"
	"github.com/news-ai/api-v1/utils"

	"github.com/news-ai/web/api"
	commonMiddleware "github.com/news-ai/web/middleware"
)

func main() {
	// Setup database & URL
	db.InitDB()
	utils.InitURL()
	err := auth.SetupAuthStore()
	if err != nil {
		log.Printf("%v", err)
		return
	}

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

	// Initialize CSRF
	// CSRF := csrf.Protect([]byte(os.Getenv("CSRFKEY")), csrf.Secure(false)) // localhost registration
	CSRF := csrf.Protect([]byte(os.Getenv("CSRFKEY")))

	// Initialize router
	router := httprouter.New()
	router.NotFound = http.HandlerFunc(api.NotFound)

	// Not found Handler
	router.GET("/", api.NotFoundHandler)
	router.GET("/api", api.NotFoundHandler)

	/*
	 * Authentication Handler
	 */

	// Login
	router.Handler("GET", "/api/auth", CSRF(auth.PasswordLoginPageHandler()))
	router.Handler("POST", "/api/auth/userlogin", CSRF(auth.PasswordLoginHandler()))

	// Forget password
	router.Handler("GET", "/api/auth/forget", CSRF(auth.ForgetPasswordPageHandler()))
	router.Handler("POST", "/api/auth/userforget", CSRF(auth.ForgetPasswordHandler()))

	// Change password
	router.Handler("GET", "/api/auth/changepassword", CSRF(auth.ChangePasswordPageHandler()))
	router.Handler("POST", "/api/auth/userchange", CSRF(auth.ChangePasswordHandler()))

	// Reset password
	router.Handler("GET", "/api/auth/resetpassword", CSRF(auth.ResetPasswordPageHandler()))
	router.Handler("POST", "/api/auth/userreset", CSRF(auth.ResetPasswordHandler()))

	// Register user
	router.Handler("GET", "/api/auth/registration", CSRF(auth.PasswordRegisterPageHandler()))
	router.Handler("POST", "/api/auth/userregister", CSRF(auth.PasswordRegisterHandler()))

	// Email confirmation
	router.Handler("GET", "/api/auth/confirmation", CSRF(auth.EmailConfirmationHandler()))

	// Invitation page
	router.Handler("GET", "/api/auth/invitation", CSRF(auth.PasswordInvitationPageHandler()))

	// Login with Google
	router.GET("/api/auth/google", auth.GoogleLoginHandler)
	router.GET("/api/auth/gmail", auth.GmailLoginHandler)
	router.GET("/api/auth/remove-gmail", auth.RemoveGmailHandler)
	router.GET("/api/auth/googlecallback", auth.GoogleCallbackHandler)

	// Login with Outlook
	router.GET("/api/auth/outlook", auth.OutlookLoginHandler)
	router.GET("/api/auth/remove-outlook", auth.RemoveOutlookHandler)
	router.GET("/api/auth/outlookcallback", auth.OutlookCallbackHandler)

	// Logout user
	router.GET("/api/auth/logout", auth.LogoutHandler)

	/*
	 * Billing Handler
	 */

	// Start a free trial
	router.Handler("GET", "/api/billing/plans/trial", auth.TrialPlanPageHandler())

	// Get all the plans
	router.Handler("GET", "/api/billing/plans", auth.ChoosePlanPageHandler())

	// Add payment method
	router.Handler("GET", "/api/billing/payment-methods", CSRF(auth.PaymentMethodsPageHandler()))
	router.Handler("POST", "/api/billing/add-payment-method", CSRF(auth.PaymentMethodsHandler()))

	// Add plan method
	router.Handler("POST", "/api/billing/confirmation", auth.ChoosePlanHandler())
	router.Handler("POST", "/api/billing/switch-confirmation", auth.ChooseSwitchPlanHandler())
	router.Handler("POST", "/api/billing/receipt", auth.ConfirmPlanHandler())

	// Cancel plan method
	router.Handler("GET", "/api/billing/cancel", CSRF(auth.CancelPlanPageHandler()))

	// Optional checks
	router.Handler("POST", "/api/billing/check-coupon", auth.CheckCouponValid())

	// Main billing page for a user
	router.Handler("GET", "/api/billing", CSRF(auth.BillingPageHandler()))

	/*
	 * API Handler
	 */

	/*
	 * General
	 */

	router.GET("/api/users", routes.UsersHandler)
	router.GET("/api/users/:id", routes.UserHandler)
	router.PATCH("/api/users/:id", routes.UserHandler)
	router.GET("/api/users/:id/:action", routes.UserActionHandler)
	router.POST("/api/users/:id/:action", routes.UserActionHandler)

	router.GET("/api/agencies", routes.AgenciesHandler)
	router.GET("/api/agencies/:id", routes.AgencyHandler)

	router.GET("/api/clients", routes.ClientsHandler)
	router.GET("/api/clients/:id", routes.ClientHandler)

	router.GET("/api/teams", routes.TeamsHandler)
	router.POST("/api/teams", routes.TeamsHandler)
	router.GET("/api/teams/:id", routes.TeamHandler)
	router.GET("/api/teams/:id/:action", routes.TeamActionHandler)

	router.GET("/api/invites", routes.InvitesHandler)
	router.POST("/api/invites", routes.InvitesHandler)

	/*
	 * Tabulae
	 */

	// router.GET("/api/publications", tabulaeRoutes.PublicationsHandler)
	// router.POST("/api/publications", tabulaeRoutes.PublicationsHandler)
	// router.GET("/api/publications/:id", tabulaeRoutes.PublicationHandler)
	// router.PATCH("/api/publications/:id", tabulaeRoutes.PublicationHandler)
	// router.GET("/api/publications/:id/:action", tabulaeRoutes.PublicationActionHandler)

	// router.GET("/api/contacts", tabulaeRoutes.ContactsHandler)
	// router.POST("/api/contacts", tabulaeRoutes.ContactsHandler)
	// router.PATCH("/api/contacts", tabulaeRoutes.ContactsHandler)
	// router.GET("/api/contacts/:id", tabulaeRoutes.ContactHandler)
	// router.PATCH("/api/contacts/:id", tabulaeRoutes.ContactHandler)
	// router.POST("/api/contacts/:id", tabulaeRoutes.ContactHandler)
	// router.DELETE("/api/contacts/:id", tabulaeRoutes.ContactHandler)
	// router.GET("/api/contacts/:id/:action", tabulaeRoutes.ContactActionHandler)

	// // router.GET("/api/contacts_v2", tabulaeRoutes.ContactsV2Handler)
	// // router.POST("/api/contacts_v2", tabulaeRoutes.ContactsV2Handler)
	// // router.GET("/api/contacts_v2/:id", tabulaeRoutes.ContactV2Handler)

	// router.GET("/api/files", tabulaeRoutes.FilesHandler)
	// router.GET("/api/files/:id", tabulaeRoutes.FileHandler)
	// router.GET("/api/files/:id/:action", tabulaeRoutes.FileActionHandler)
	// router.POST("/api/files/:id/:action", tabulaeRoutes.FileActionHandler)

	// router.GET("/api/lists", tabulaeRoutes.MediaListsHandler)
	// router.POST("/api/lists", tabulaeRoutes.MediaListsHandler)
	// router.GET("/api/lists/:id", tabulaeRoutes.MediaListHandler)
	// router.PATCH("/api/lists/:id", tabulaeRoutes.MediaListHandler)
	// router.DELETE("/api/lists/:id", tabulaeRoutes.MediaListHandler)
	// router.GET("/api/lists/:id/:action", tabulaeRoutes.MediaListActionHandler)
	// router.POST("/api/lists/:id/:action", tabulaeRoutes.MediaListActionHandler)

	// router.GET("/api/emails", tabulaeRoutes.EmailsHandler)
	// router.POST("/api/emails", tabulaeRoutes.EmailsHandler)
	// router.PATCH("/api/emails", tabulaeRoutes.EmailsHandler)
	// router.GET("/api/emails/:id", tabulaeRoutes.EmailHandler)
	// router.PATCH("/api/emails/:id", tabulaeRoutes.EmailHandler)
	// router.POST("/api/emails/:id", tabulaeRoutes.EmailHandler)
	// router.GET("/api/emails/:id/:action", tabulaeRoutes.EmailActionHandler)
	// router.POST("/api/emails/:id/:action", tabulaeRoutes.EmailActionHandler)

	// router.GET("/api/email-settings", tabulaeRoutes.EmailSettingsHandler)
	// router.POST("/api/email-settings", tabulaeRoutes.EmailSettingsHandler)
	// router.GET("/api/email-settings/:id", tabulaeRoutes.EmailSettingHandler)
	// router.POST("/api/email-settings/:id", tabulaeRoutes.EmailSettingHandler)
	// router.GET("/api/email-settings/:id/:action", tabulaeRoutes.EmailSettingActionHandler)

	// router.GET("/api/templates", tabulaeRoutes.TemplatesHandler)
	// router.POST("/api/templates", tabulaeRoutes.TemplatesHandler)
	// router.GET("/api/templates/:id", tabulaeRoutes.TemplateHandler)
	// router.PATCH("/api/templates/:id", tabulaeRoutes.TemplateHandler)

	// router.GET("/api/feeds", tabulaeRoutes.FeedsHandler)
	// router.POST("/api/feeds", tabulaeRoutes.FeedsHandler)
	// router.GET("/api/feeds/:id", tabulaeRoutes.FeedHandler)
	// router.DELETE("/api/feeds/:id", tabulaeRoutes.FeedHandler)

	// Security fixes
	secureMiddleware := secure.New(secure.Options{
		FrameDeny:        true,
		BrowserXssFilter: true,
	})

	app.Use(negroni.HandlerFunc(middleware.UpdateOrCreateUser))
	app.Use(negroni.HandlerFunc(commonMiddleware.AttachParameters))
	app.Use(negroni.HandlerFunc(secureMiddleware.HandlerFuncWithNext))

	app.UseHandler(router)

	// Register the app router
	http.Handle("/", context.ClearHandler(app))
	http.ListenAndServe(":8080", nil)
}
