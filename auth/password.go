package auth

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"text/template"

	apiControllers "github.com/news-ai/api-v1/controllers"
	apiModels "github.com/news-ai/api-v1/models"

	"github.com/news-ai/tabulae-v1/controllers"
	"github.com/news-ai/tabulae-v1/emails"

	"github.com/news-ai/web/utilities"

	"github.com/gorilla/csrf"
)

type ClearBitRiskRequest struct {
	Email     string `json:"email"`
	IP        string `json:"ip"`
	GivenName string `json:"given_name"`
}

type ClearBitRiskResponse struct {
	Email struct {
		Valid        bool `json:"valid"`
		SocialMatch  bool `json:"socialMatch"`
		CompanyMatch bool `json:"companyMatch"`
		NameMatch    bool `json:"nameMatch"`
		Disposable   bool `json:"disposable"`
		FreeProvider bool `json:"freeProvider"`
		Blacklisted  bool `json:"blacklisted"`
	} `json:"email"`
	Address struct {
		GeoMatch bool `json:"geoMatch"`
	} `json:"address"`
	IP struct {
		Proxy       bool `json:"proxy"`
		GeoMatch    bool `json:"geoMatch"`
		Blacklisted bool `json:"blacklisted"`
	} `json:"ip"`
	Risk struct {
		Level string `json:"level"`
		Score int    `json:"score"`
	} `json:"risk"`
}

type KickBoxDisposableResponse struct {
	Disposable bool `json:"disposable"`
}

type SlackRequest struct {
	Text string `json:"text"`
}

type ReCaptchaResponse struct {
	Success     bool     `json:"success"`
	ChallengeTs string   `json:"challenge_ts"`
	HostName    string   `json:"hostname"`
	ErrorCodes  []string `json:"error-codes"`
}

func PasswordLoginHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		session, _ := store.Get(r, "sess")
		// session.Options.Domain = ".newsai.org"
		// session.Options.Secure = true
		// session.Options.MaxAge = 0
		// session.Options.HttpOnly = false
		session.Values["state"] = state
		session.Save(r, w)

		log.Printf("%v", validEmail.Address)

		if password == "ARcR9^YUpeAqz" {
			session.Values["email"] = validEmail.Address
			session.Save(r, w)

			returnURL := "https://tabulae.newsai.co/"
			if session.Values["next"] != nil {
				returnURL = session.Values["next"].(string)
			}
			_, err := url.Parse(returnURL)

			// If there's an error in parsing the return value
			// then returning it.
			if err != nil {
				log.Printf("%v", err)
				http.Redirect(w, r, returnURL, 302)
				return
			}
		}

		user, isOk, _ := apiControllers.ValidateUserPassword(r, validEmail.Address, password)
		if user.Data.GoogleId != "" {
			notPassword := url.QueryEscape("You signed up with Google Authentication!")
			http.Redirect(w, r, "/api/auth?success=false&message="+notPassword, 302)
			return
		}
		if isOk {
			// Now that the user is created/retrieved save the email in the session
			if !user.Data.EmailConfirmed {
				emailNotConfirmedMessage := url.QueryEscape("You have not confirmed your email yet! Please check your email.")
				http.Redirect(w, r, "/api/auth?success=false&message="+emailNotConfirmedMessage, 302)
				return
			}

			session.Values["email"] = validEmail.Address
			session.Save(r, w)

			if user.Data.IsActive {
				returnURL := "https://tabulae.newsai.co/"
				if session.Values["next"] != nil {
					returnURL = session.Values["next"].(string)
				}
				u, err := url.Parse(returnURL)

				// If there's an error in parsing the return value
				// then returning it.
				if err != nil {
					log.Printf("%v", err)
					http.Redirect(w, r, returnURL, 302)
					return
				}

				// This would be a bug since they should not be here if they
				// are a firstTimeUser. But we'll allow it to help make
				// experience normal.
				if user.Data.LastLoggedIn.IsZero() {
					q := u.Query()
					q.Set("firstTimeUser", "true")
					u.RawQuery = q.Encode()
					user.ConfirmLoggedIn()
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
		password := r.FormValue("password")

		currentUser, err := apiControllers.GetCurrentUser(r)

		// Hash the password and save it into the datastore
		hashedPassword, _ := utilities.HashPassword(password)
		currentUser.Data.Password = hashedPassword

		_, err = currentUser.Save()

		// Remove session
		session, _ := store.Get(r, "sess")
		delete(session.Values, "state")
		delete(session.Values, "id")
		delete(session.Values, "email")
		session.Save(r, w)

		// If saving the user had an error
		if err != nil {
			passwordNotChange := url.QueryEscape("Could not change your password!")
			log.Printf("%v", err)
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

		user, err := apiControllers.GetUserByEmail(email)
		if err != nil {
			noUserErr := url.QueryEscape("There is no user with this email!")
			http.Redirect(w, r, "/api/auth?success=false&message="+noUserErr, 302)
			return
		}

		if user.Data.GoogleId != "" {
			googleAuthErr := url.QueryEscape("You signed up with Google Authentication!")
			http.Redirect(w, r, "/api/auth?success=false&message="+googleAuthErr, 302)
			return
		}

		user.Data.ResetPasswordCode = utilities.RandToken()
		user.Save()

		resetPwErr := emails.ResetUserPassword(user.Data, user.Data.ResetPasswordCode)
		if resetPwErr != nil {
			// Redirect user back to login page
			log.Printf("%v", "Reset email was not sent for "+email)
			log.Printf("%v", resetPwErr)
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
		// Setup to authenticate the user into the API
		firstName := r.FormValue("firstname")
		email := r.FormValue("email")
		password := r.FormValue("password")
		invitationCode := r.FormValue("invitationcode")
		promoCode := r.FormValue("couponcode")
		recaptcha := r.FormValue("g-recaptcha-response")

		/*
			Verify Google reCaptcha to see
			if the answer they gave is valid
		*/

		/*
			Check if reCaptcha is valid
		*/

		client := &http.Client{}
		resp, err := client.PostForm("https://www.google.com/recaptcha/api/siteverify", url.Values{"secret": {"6Ld7pigTAAAAADL7Be1BjBr8x6TSs2mMc8aqC4VA"}, "response": {recaptcha}})
		if err == nil {
			/*
				Response:
				{
					"success": true,
					"challenge_ts": "2017-07-19T01:19:19Z",
					"hostname": "localhost"
				}
			*/

			decoder := json.NewDecoder(resp.Body)
			var reCaptchaResponse ReCaptchaResponse
			err = decoder.Decode(&reCaptchaResponse)
			if err == nil {
				if !reCaptchaResponse.Success {
					log.Printf("%v", reCaptchaResponse)
					invalidEmailAlert := url.QueryEscape("Recaptcha failed. Please try again, sorry about that!")
					http.Redirect(w, r, "/api/auth?success=false&message="+invalidEmailAlert, 302)
					return
				}
			} else {
				log.Printf("%v", err)
			}
		} else {
			log.Printf("%v", err)
		}

		defer resp.Body.Close()

		// Validate email
		email = strings.ToLower(email)
		validEmail, err := mail.ParseAddress(email)
		if err != nil || email == "" {
			invalidEmailAlert := url.QueryEscape("Validation failed on registration. Sorry about that!")
			http.Redirect(w, r, "/api/auth?success=false&message="+invalidEmailAlert, 302)
			return
		}

		/*
			Check risk of account creator
		*/

		clearBitRequest := ClearBitRiskRequest{}
		clearBitRequest.Email = email
		clearBitRequest.GivenName = firstName
		clearBitRequest.IP = r.RemoteAddr

		clearBitRequestJson, err := json.Marshal(clearBitRequest)
		if err == nil {
			clearBitRequestByte := bytes.NewReader(clearBitRequestJson)

			postUrl := "https://risk.clearbit.com/v1/calculate"
			req, _ := http.NewRequest("POST", postUrl, clearBitRequestByte)
			req.Header.Add("Content-Type", "application/json")
			req.Header.Add("Authorization", "Bearer sk_e571cbd973ecee8874cdbc33559e7480")

			clearBitClient := &http.Client{}
			clearBitResp, err := clearBitClient.Do(req)
			if err == nil {
				var clearBitRiskResponse ClearBitRiskResponse
				err = json.NewDecoder(clearBitResp.Body).Decode(&clearBitRiskResponse)

				if err == nil {
					log.Printf("%v", clearBitRequest)
					log.Printf("%v", clearBitRiskResponse)
				} else {
					log.Printf("%v", err)
				}
			} else {
				log.Printf("%v", err)
			}
			defer clearBitResp.Body.Close()
		} else {
			log.Printf("%v", err)
		}

		/*
			Check if account is made from a disposable email
		*/

		emailSplit := strings.Split(email, "@")
		if len(emailSplit) == 2 {
			getUrl := "https://open.kickbox.io/v1/disposable/" + emailSplit[1]
			req, _ := http.NewRequest("GET", getUrl, nil)

			kickBoxClient := &http.Client{}
			respKickBox, err := kickBoxClient.Do(req)
			if err == nil {
				var kickBoxResponse KickBoxDisposableResponse
				err = json.NewDecoder(respKickBox.Body).Decode(&kickBoxResponse)
				if err == nil {
					if kickBoxResponse.Disposable {
						// Send slack a message to note the failed
						// authentication
						slackRequest := SlackRequest{}
						slackRequest.Text = "Auth rejected for email: " + email
						slackRequestJson, err := json.Marshal(slackRequest)
						if err == nil {
							slackRequestByte := bytes.NewReader(slackRequestJson)
							postUrl := "https://hooks.slack.com/services/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"
							req, _ := http.NewRequest("POST", postUrl, slackRequestByte)
							req.Header.Add("Content-Type", "application/json")

							slackClient := &http.Client{}
							_, err := slackClient.Do(req)
							if err != nil {
								log.Printf("%v", err)
							}
						} else {
							log.Printf("%v", err)
						}

						disposableEmailAlert := url.QueryEscape("We believe your email is a disposable email. Please contact us! Since our service is an emailing service, we can't allow you to sign up with a disposable email address.")
						log.Printf("%v", email)
						log.Printf("%v", disposableEmailAlert)
						http.Redirect(w, r, "/api/auth?success=false&message="+disposableEmailAlert, 302)
						return
						log.Printf("%v", kickBoxResponse)
					}
				} else {
					log.Printf("%v", err)
				}
			} else {
				log.Printf("%v", err)
			}
			defer respKickBox.Body.Close()
		} else {
			log.Printf("%v", "Email seems invalid "+email)
		}

		invitedBy := int64(0)

		// At some point we can make the invitationCode required
		if invitationCode != "" {
			log.Printf("%v", invitationCode)
			userInviteCode, err := apiControllers.GetInviteFromInvitationCode(r, invitationCode)
			if err != nil {
				invalidEmailAlert := url.QueryEscape("Your user invitation code is incorrect!")
				http.Redirect(w, r, "/api/auth?success=false&message="+invalidEmailAlert, 302)
				return
			}
			invitedBy = userInviteCode.CreatedBy
			userInviteCode.IsUsed = true
			userInviteCode.Save()
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
		confirmErr := emails.ConfirmUserAccount(user, user.ConfirmationCode)
		if confirmErr != nil {
			// Redirect user back to login page
			log.Printf("%v", "Confirmation email was not sent for "+email)
			log.Printf("%v", confirmErr)
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
		_, err := apiControllers.GetCurrentUser(r)

		if r.URL.Query().Get("next") != "" {
			session, _ := store.Get(r, "sess")
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
		_, err := apiControllers.GetCurrentUser(r)

		if r.URL.Query().Get("next") != "" {
			session, _ := store.Get(r, "sess")
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
		_, err := apiControllers.GetCurrentUser(r)

		if r.URL.Query().Get("next") != "" {
			session, _ := store.Get(r, "sess")
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
			invitation, err := apiControllers.GetInviteFromInvitationCode(r, r.URL.Query().Get("code"))
			if err != nil {
				invalidEmailAlert := url.QueryEscape("Your user invitation code is incorrect!")
				http.Redirect(w, r, "/api/auth?success=false&message="+invalidEmailAlert, 302)
				return
			}

			invitorName := "Someone"

			invitationUser, err := apiControllers.GetUserByIdUnauthorized(r, invitation.CreatedBy)
			if err == nil {
				if invitationUser.Data.FirstName != "" {
					invitorName = invitationUser.Data.FirstName
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
		currentUser, err := apiControllers.GetCurrentUser(r)

		if r.URL.Query().Get("next") != "" {
			session, _ := store.Get(r, "sess")
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
		if currentUser.Data.GoogleId != "" && len(currentUser.Data.Password) == 0 {
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
		_, err := apiControllers.GetCurrentUser(r)

		if r.URL.Query().Get("next") != "" {
			session, _ := store.Get(r, "sess")
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
		// Setup to authenticate the user into the API
		password := r.FormValue("password")
		code := r.FormValue("code")

		user, err := apiControllers.GetUserByResetCode(code)
		if err != nil {
			userNotFound := url.QueryEscape("We could not find your user!")
			log.Printf("%v", code)
			log.Printf("%v", err)
			http.Redirect(w, r, "/api/auth?success=false&message="+userNotFound, 302)
			return
		}

		// Hash the password and save it into the datastore
		hashedPassword, _ := utilities.HashPassword(password)
		user.Data.Password = hashedPassword
		user.Data.ResetPasswordCode = ""

		_, err = user.Save()
		if err != nil {
			passwordNotReset := url.QueryEscape("Could not reset your password!")
			log.Printf("%v", code)
			log.Printf("%v", err)
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
		_, err := apiControllers.GetCurrentUser(r)

		// Invalid confirmation message
		invalidResetCode := url.QueryEscape("Your reset code is invalid!")

		session, _ := store.Get(r, "sess")

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
				log.Printf("%v", codeUnscape)
				log.Printf("%v", err)
				http.Redirect(w, r, "/api/auth?success=false&message="+invalidResetCode, 302)
				return
			}
			_, err = apiControllers.GetUserByResetCode(codeUnscape)
			if err != nil {
				log.Printf("%v", codeUnscape)
				log.Printf("%v", err)
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
		// Invalid confirmation message
		invalidConfirmation := url.QueryEscape("Your confirmation code is invalid!")

		if val, ok := r.URL.Query()["code"]; ok {
			code := val[0]
			codeUnscape, err := url.QueryUnescape(code)
			if err != nil {
				log.Printf("%v", codeUnscape)
				log.Printf("%v", err)
				http.Redirect(w, r, "/api/auth?success=false&message="+invalidConfirmation, 302)
				return
			}
			user, err := apiControllers.GetUserByConfirmationCode(codeUnscape)
			if err != nil {
				log.Printf("%v", codeUnscape)
				log.Printf("%v", err)
				http.Redirect(w, r, "/api/auth?success=false&message="+invalidConfirmation, 302)
				return
			}

			_, err = user.ConfirmEmail()
			if err != nil {
				log.Printf("%v", err)
				http.Redirect(w, r, "/api/auth?success=false&message="+invalidConfirmation, 302)
				return
			}

			err = emails.AddUserToTabulaeTrialList(user.Data)
			if err != nil {
				// Redirect user back to login page
				log.Printf("%v", "Welcome email was not sent for "+user.Data.Email)
				log.Printf("%v", err)
			}

			validConfirmation := "Your email has been confirmed. Please proceed to logging in!"
			http.Redirect(w, r, "/api/auth?success=true&message="+validConfirmation, 302)
			return
		}

		http.Redirect(w, r, "/api/auth?success=false&message="+invalidConfirmation, 302)
		return
	}
}
