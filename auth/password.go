package auth

import (
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"text/template"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"

	apiControllers "github.com/news-ai/api/controllers"
	apiModels "github.com/news-ai/api/models"

	"github.com/news-ai/tabulae/controllers"
	"github.com/news-ai/tabulae/emails"

	"github.com/news-ai/web/utilities"

	"github.com/gorilla/csrf"
)

func PasswordLoginHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := appengine.NewContext(r)
		// Setup to authenticate the user into the API
		email := r.FormValue("email")
		password := r.FormValue("password")

		email = strings.ToLower(email)

		// Validate email
		validEmail, err := mail.ParseAddress(email)
		if err != nil {
			invalidEmailAlert := url.QueryEscape("The email you entered is not valid!")
			http.Redirect(w, r, "/api/auth?success=false&message="+invalidEmailAlert, 302)
			return
		}

		// Generate a random state that we identify the user with
		state := utilities.RandToken()

		// Save the session for each of the users
		session, _ := Store.Get(r, "sess")
		session.Values["state"] = state
		session.Save(r, w)

		log.Infof(c, "%v", validEmail.Address)

		user, isOk, _ := apiControllers.ValidateUserPassword(r, validEmail.Address, password)
		if isOk {
			if user.GoogleId != "" {
				notPassword := url.QueryEscape("You signed up with Google Authentication!")
				http.Redirect(w, r, "/api/auth?success=false&message="+notPassword, 302)
				return
			}
			// // Now that the user is created/retrieved save the email in the session
			if !user.EmailConfirmed {
				emailNotConfirmedMessage := url.QueryEscape("You have not confirmed your email yet! Please check your email.")
				http.Redirect(w, r, "/api/auth?success=false&message="+emailNotConfirmedMessage, 302)
				return
			}

			session.Values["email"] = validEmail.Address
			session.Save(r, w)

			if user.IsActive {
				returnURL := "https://tabulae.newsai.co/"
				if session.Values["next"] != nil {
					returnURL = session.Values["next"].(string)
				}
				u, err := url.Parse(returnURL)

				// If there's an error in parsing the return value
				// then returning it.
				if err != nil {
					log.Errorf(c, "%v", err)
					http.Redirect(w, r, returnURL, 302)
					return
				}

				// This would be a bug since they should not be here if they
				// are a firstTimeUser. But we'll allow it to help make
				// experience normal.
				if user.LastLoggedIn.IsZero() {
					q := u.Query()
					q.Set("firstTimeUser", "true")
					u.RawQuery = q.Encode()
					user.ConfirmLoggedIn(c)
				}
				http.Redirect(w, r, u.String(), 302)
				return
			} else {
				http.Redirect(w, r, "/api/billing/plans/trial", 302)
				return
			}
		}

		wrongPasswordMessage := url.QueryEscape("You entered the wrong password!")
		http.Redirect(w, r, "/api/auth?success=false&message="+wrongPasswordMessage, 302)
		return
	}
}

func ChangePasswordHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := appengine.NewContext(r)
		password := r.FormValue("password")

		currentUser, err := apiControllers.GetCurrentUser(c, r)

		// Hash the password and save it into the datastore
		hashedPassword, _ := utilities.HashPassword(password)
		currentUser.Password = hashedPassword

		_, err = currentUser.Save(c)

		// Remove session
		session, _ := Store.Get(r, "sess")
		delete(session.Values, "state")
		delete(session.Values, "id")
		delete(session.Values, "email")
		session.Save(r, w)

		// If saving the user had an error
		if err != nil {
			passwordNotChange := url.QueryEscape("Could not change your password!")
			log.Infof(c, "%v", err)
			http.Redirect(w, r, "/api/auth?success=false&message="+passwordNotChange, 302)
			return
		}

		// If password is changed
		validChange := "Your password has been changed! Please login with your new password."
		http.Redirect(w, r, "/api/auth?success=true&message="+validChange, 302)
		return
	}
}

func ForgetPasswordHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := appengine.NewContext(r)
		// Forget password
		email := r.FormValue("email")
		email = strings.ToLower(email)

		// Validate email
		_, err := mail.ParseAddress(email)
		if err != nil {
			invalidEmailAlert := url.QueryEscape("The email you entered is not valid!")
			http.Redirect(w, r, "/api/auth?success=false&message="+invalidEmailAlert, 302)
			return
		}

		user, err := apiControllers.GetUserByEmail(c, email)
		if err != nil {
			noUserErr := url.QueryEscape("There is no user with this email!")
			http.Redirect(w, r, "/api/auth?success=false&message="+noUserErr, 302)
			return
		}

		if user.GoogleId != "" {
			googleAuthErr := url.QueryEscape("You signed up with Google Authentication!")
			http.Redirect(w, r, "/api/auth?success=false&message="+googleAuthErr, 302)
			return
		}

		user.ResetPasswordCode = utilities.RandToken()
		user.Save(c)

		resetPwErr := emails.ResetUserPassword(c, user, user.ResetPasswordCode)
		if resetPwErr != nil {
			// Redirect user back to login page
			log.Errorf(c, "%v", "Reset email was not sent for "+email)
			log.Errorf(c, "%v", resetPwErr)
			emailResetErr := url.QueryEscape("Could not send a reset email. We'll fix this soon!")
			http.Redirect(w, r, "/api/auth?success=false&message="+emailResetErr, 302)
			return
		}

		// Redirect user back to login page
		resetMessage := url.QueryEscape("We sent you a password reset email!")
		http.Redirect(w, r, "/api/auth?success=true&message="+resetMessage, 302)
		return
	}
}

// Don't start their session here, but when they login to the platform.
// This is just to give them the ability to register an account.
func PasswordRegisterHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := appengine.NewContext(r)
		// Setup to authenticate the user into the API
		firstName := r.FormValue("firstname")
		email := r.FormValue("email")
		password := r.FormValue("password")
		invitationCode := r.FormValue("invitationcode")
		promoCode := r.FormValue("couponcode")

		email = strings.ToLower(email)

		// Validate email
		validEmail, err := mail.ParseAddress(email)
		if err != nil || email == "" {
			invalidEmailAlert := url.QueryEscape("Validation failed on registration. Sorry about that!")
			http.Redirect(w, r, "/api/auth?success=false&message="+invalidEmailAlert, 302)
			return
		}

		if strings.Contains(email, "@qiq.us") || strings.Contains(email, "@10vpn.info") {
			invalidEmailAlert := url.QueryEscape("Validation failed on registration. Sorry about that!")
			http.Redirect(w, r, "/api/auth?success=false&message="+invalidEmailAlert, 302)
			return
		}

		invitedBy := int64(0)

		// At some point we can make the invitationCode required
		if invitationCode != "" {
			log.Infof(c, "%v", invitationCode)
			userInviteCode, err := apiControllers.GetInviteFromInvitationCode(c, r, invitationCode)
			if err != nil {
				invalidEmailAlert := url.QueryEscape("Your user invitation code is incorrect!")
				http.Redirect(w, r, "/api/auth?success=false&message="+invalidEmailAlert, 302)
				return
			}
			invitedBy = userInviteCode.CreatedBy
			userInviteCode.IsUsed = true
			userInviteCode.Save(c)
		}

		// Hash the password and save it into the datastore
		hashedPassword, _ := utilities.HashPassword(password)

		user := apiModels.User{}
		user.FirstName = firstName
		user.Email = validEmail.Address
		user.Password = hashedPassword
		user.EmailConfirmed = false
		user.AgreeTermsAndConditions = true
		user.ConfirmationCode = utilities.RandToken()
		user.InvitedBy = invitedBy // Potentially also email the person who invited them
		user.IsActive = false
		user.PromoCode = promoCode

		// Register user
		_, isOk, err := controllers.RegisterUser(r, user)

		if !isOk && err != nil {
			// Redirect user back to login page
			emailRegistered := url.QueryEscape("Email has already been registered")
			http.Redirect(w, r, "/api/auth?success=false&message="+emailRegistered, 302)
			return
		}

		// Email could fail to send if there is no singleUser. Create check later.
		confirmErr := emails.ConfirmUserAccount(c, user, user.ConfirmationCode)
		if confirmErr != nil {
			// Redirect user back to login page
			log.Errorf(c, "%v", "Confirmation email was not sent for "+email)
			log.Errorf(c, "%v", confirmErr)
			emailRegistered := url.QueryEscape("Could not send confirmation email. We'll fix this soon!")
			http.Redirect(w, r, "/api/auth?success=false&message="+emailRegistered, 302)
			return
		}

		// Redirect user back to login page
		confirmationMessage := url.QueryEscape("We sent you a confirmation email!")
		http.Redirect(w, r, "/api/auth?success=true&message="+confirmationMessage, 302)
		return
	}
}

// Takes ?next as well. Create a session for the person.
// Will post data to the password login handler.
// Redirect to the ?next parameter.
// Put CSRF token into the login handler.
func PasswordLoginPageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := appengine.NewContext(r)
		_, err := apiControllers.GetCurrentUser(c, r)

		if r.URL.Query().Get("next") != "" {
			session, _ := Store.Get(r, "sess")
			session.Values["next"] = r.URL.Query().Get("next")
			session.Save(r, w)

			// If there is a next and the user has been logged in
			if err == nil {
				http.Redirect(w, r, session.Values["next"].(string), 302)
				return
			}
		}

		// If there is no next and the user is logged in
		if err == nil {
			http.Redirect(w, r, "https://tabulae.newsai.co/", 302)
			return
		}

		// If there is no user then we redirect them to the login page
		t := template.New("login.html")
		t, err = t.ParseFiles("auth/login.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data := map[string]interface{}{
			csrf.TemplateTag: csrf.TemplateField(r),
		}

		t.Execute(w, data)
		return
	}
}

// You have to be logged out in order to register a new user
func PasswordRegisterPageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := appengine.NewContext(r)
		_, err := apiControllers.GetCurrentUser(c, r)

		if r.URL.Query().Get("next") != "" {
			session, _ := Store.Get(r, "sess")
			session.Values["next"] = r.URL.Query().Get("next")
			session.Save(r, w)

			// If there is a next and the user has been logged in
			if err == nil {
				http.Redirect(w, r, r.URL.Query().Get("next"), 302)
				return
			}
		}

		// If there is no next and the user is logged in
		if err == nil {
			http.Redirect(w, r, "https://tabulae.newsai.co/", 302)
			return
		}

		data := map[string]interface{}{
			csrf.TemplateTag: csrf.TemplateField(r),
		}

		t := template.New("register.html")
		t, _ = t.ParseFiles("auth/register.html")
		t.Execute(w, data)
	}
}

// Invitation
func PasswordInvitationPageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := appengine.NewContext(r)
		_, err := apiControllers.GetCurrentUser(c, r)

		if r.URL.Query().Get("next") != "" {
			session, _ := Store.Get(r, "sess")
			session.Values["next"] = r.URL.Query().Get("next")
			session.Save(r, w)

			// If there is a next and the user has been logged in
			if err == nil {
				http.Redirect(w, r, r.URL.Query().Get("next"), 302)
				return
			}
		}

		// If there is no next and the user is logged in
		if err == nil {
			http.Redirect(w, r, "https://tabulae.newsai.co/", 302)
			return
		}

		// Invitation code
		if r.URL.Query().Get("code") != "" {
			invitation, err := apiControllers.GetInviteFromInvitationCode(c, r, r.URL.Query().Get("code"))
			if err != nil {
				invalidEmailAlert := url.QueryEscape("Your user invitation code is incorrect!")
				http.Redirect(w, r, "/api/auth?success=false&message="+invalidEmailAlert, 302)
				return
			}

			invitorName := "Someone"

			invitationUser, err := apiControllers.GetUserByIdUnauthorized(c, r, invitation.CreatedBy)
			if err == nil {
				if invitationUser.FirstName != "" {
					invitorName = invitationUser.FirstName
				}
			}

			data := map[string]interface{}{
				csrf.TemplateTag: csrf.TemplateField(r),
				"invitorName":    invitorName,
			}

			t := template.New("invitation.html")
			t, _ = t.ParseFiles("auth/invitation.html")
			t.Execute(w, data)
		} else {
			invalidInvitationCode := url.QueryEscape("The invitation code you have entered is invalid.")
			http.Redirect(w, r, "/api/auth?success=false&message="+invalidInvitationCode, 302)
			return
		}
	}
}

func ChangePasswordPageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := appengine.NewContext(r)
		currentUser, err := apiControllers.GetCurrentUser(c, r)

		if r.URL.Query().Get("next") != "" {
			session, _ := Store.Get(r, "sess")
			session.Values["next"] = r.URL.Query().Get("next")
			session.Save(r, w)

			// If there is a next and the user has not been logged in
			if err != nil {
				http.Redirect(w, r, r.URL.Query().Get("next"), 302)
				return
			}
		}

		// If there is no next and the user is not logged in
		if err != nil {
			http.Redirect(w, r, "https://tabulae.newsai.co/", 302)
			return
		}

		// If uses Google authentication and there is no password
		if currentUser.GoogleId != "" && len(currentUser.Password) == 0 {
			http.Redirect(w, r, "https://tabulae.newsai.co/", 302)
			return
		}

		data := map[string]interface{}{
			csrf.TemplateTag: csrf.TemplateField(r),
		}

		t := template.New("change.html")
		t, _ = t.ParseFiles("profile/change.html")
		t.Execute(w, data)
	}
}

func ForgetPasswordPageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := appengine.NewContext(r)
		_, err := apiControllers.GetCurrentUser(c, r)

		if r.URL.Query().Get("next") != "" {
			session, _ := Store.Get(r, "sess")
			session.Values["next"] = r.URL.Query().Get("next")
			session.Save(r, w)

			// If there is a next and the user has been logged in
			if err == nil {
				http.Redirect(w, r, r.URL.Query().Get("next"), 302)
				return
			}
		}

		// If there is no next and the user is logged in
		if err == nil {
			http.Redirect(w, r, "https://tabulae.newsai.co/", 302)
			return
		}

		data := map[string]interface{}{
			csrf.TemplateTag: csrf.TemplateField(r),
		}

		t := template.New("forget.html")
		t, _ = t.ParseFiles("auth/forget.html")
		t.Execute(w, data)
	}
}

func ResetPasswordHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := appengine.NewContext(r)
		// Setup to authenticate the user into the API
		password := r.FormValue("password")
		code := r.FormValue("code")

		user, err := apiControllers.GetUserByResetCode(c, code)
		if err != nil {
			userNotFound := url.QueryEscape("We could not find your user!")
			log.Infof(c, "%v", code)
			log.Infof(c, "%v", err)
			http.Redirect(w, r, "/api/auth?success=false&message="+userNotFound, 302)
			return
		}

		// Hash the password and save it into the datastore
		hashedPassword, _ := utilities.HashPassword(password)
		user.Password = hashedPassword
		user.ResetPasswordCode = ""

		_, err = user.Save(c)
		if err != nil {
			passwordNotReset := url.QueryEscape("Could not reset your password!")
			log.Infof(c, "%v", code)
			log.Infof(c, "%v", err)
			http.Redirect(w, r, "/api/auth?success=false&message="+passwordNotReset, 302)
			return
		}

		validReset := "Your password has been changed!"
		http.Redirect(w, r, "/api/auth?success=true&message="+validReset, 302)
		return
	}
}

func ResetPasswordPageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := appengine.NewContext(r)
		_, err := apiControllers.GetCurrentUser(c, r)

		// Invalid confirmation message
		invalidResetCode := url.QueryEscape("Your reset code is invalid!")

		session, _ := Store.Get(r, "sess")

		if r.URL.Query().Get("next") != "" {
			session.Values["next"] = r.URL.Query().Get("next")
			session.Save(r, w)

			// If there is a next and the user has been logged in
			if err == nil {
				http.Redirect(w, r, r.URL.Query().Get("next"), 302)
				return
			}
		}

		// If there is no next and the user is logged in
		if err == nil {
			http.Redirect(w, r, "https://tabulae.newsai.co/", 302)
			return
		}

		// Validate token
		if val, ok := r.URL.Query()["code"]; ok {
			code := val[0]
			codeUnscape, err := url.QueryUnescape(code)
			if err != nil {
				log.Infof(c, "%v", codeUnscape)
				log.Infof(c, "%v", err)
				http.Redirect(w, r, "/api/auth?success=false&message="+invalidResetCode, 302)
				return
			}
			_, err = apiControllers.GetUserByResetCode(c, codeUnscape)
			if err != nil {
				log.Infof(c, "%v", codeUnscape)
				log.Infof(c, "%v", err)
				http.Redirect(w, r, "/api/auth?success=false&message="+invalidResetCode, 302)
				return
			}
			session.Values["resetCode"] = codeUnscape
			session.Save(r, w)
		} else {
			// If there is no reset code then return to the login page
			noResetCode := url.QueryEscape("There is no reset code provided!")
			http.Redirect(w, r, "/api/auth?success=false&message="+noResetCode, 302)
			return
		}

		data := map[string]interface{}{
			csrf.TemplateTag: csrf.TemplateField(r),
		}

		t := template.New("reset.html")
		t, _ = t.ParseFiles("auth/reset.html")
		t.Execute(w, data)
		return
	}
}

func EmailConfirmationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := appengine.NewContext(r)

		// Invalid confirmation message
		invalidConfirmation := url.QueryEscape("Your confirmation code is invalid!")

		if val, ok := r.URL.Query()["code"]; ok {
			code := val[0]
			codeUnscape, err := url.QueryUnescape(code)
			if err != nil {
				log.Infof(c, "%v", codeUnscape)
				log.Infof(c, "%v", err)
				http.Redirect(w, r, "/api/auth?success=false&message="+invalidConfirmation, 302)
				return
			}
			user, err := apiControllers.GetUserByConfirmationCode(c, codeUnscape)
			if err != nil {
				log.Infof(c, "%v", codeUnscape)
				log.Infof(c, "%v", err)
				http.Redirect(w, r, "/api/auth?success=false&message="+invalidConfirmation, 302)
				return
			}

			_, err = user.ConfirmEmail(c)
			if err != nil {
				log.Infof(c, "%v", err)
				http.Redirect(w, r, "/api/auth?success=false&message="+invalidConfirmation, 302)
				return
			}

			err = emails.AddUserToTabulaeTrialList(c, user)
			if err != nil {
				// Redirect user back to login page
				log.Errorf(c, "%v", "Welcome email was not sent for "+user.Email)
				log.Errorf(c, "%v", err)
			}

			validConfirmation := "Your email has been confirmed. Please proceed to logging in!"
			http.Redirect(w, r, "/api/auth?success=true&message="+validConfirmation, 302)
			return
		}

		http.Redirect(w, r, "/api/auth?success=false&message="+invalidConfirmation, 302)
		return
	}
}
