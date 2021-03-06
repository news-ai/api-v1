package controllers

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"time"

	gcontext "github.com/gorilla/context"
	"github.com/pquerna/ffjson/ffjson"

	"github.com/news-ai/api-v1/billing"
	"github.com/news-ai/api-v1/db"
	"github.com/news-ai/api-v1/models"

	"github.com/news-ai/tabulae-v1/emails"

	"github.com/news-ai/web/permissions"
	"github.com/news-ai/web/utilities"
)

/*
* Private methods
 */

/*
* Get methods
 */

func getUser(r *http.Request, id int64) (models.UserPostgres, error) {
	// Get the current signed in user details by Id
	postgresUser := models.UserPostgres{}
	err := db.DB.Model(&postgresUser).Where("id = ?", id).Select()
	if err != nil {
		log.Printf("%v", err)
		return models.UserPostgres{}, err
	}

	if postgresUser.Data.Email != "" {
		// postgresUser.Format(userId, "users")
		currentUser, err := GetCurrentUser(r)
		if err != nil {
			log.Printf("%v", err)
			return models.UserPostgres{}, err
		}

		if postgresUser.Data.TeamId != currentUser.Data.TeamId && !permissions.AccessToObject(postgresUser.Data.Id, currentUser.Data.Id) && !currentUser.Data.IsAdmin {
			err = errors.New("Forbidden")
			log.Printf("%v", err)
			return models.UserPostgres{}, err
		}

		postgresUser.Data.Type = "users"
		postgresUser.Data.Id = postgresUser.Id
		return postgresUser, nil
	}

	return models.UserPostgres{}, errors.New("No user by this id")
}

// Gets every single user
func getUsers(r *http.Request) ([]models.UserPostgres, error) {
	user, err := GetCurrentUser(r)
	if err != nil {
		log.Printf("%v", err)
		return []models.UserPostgres{}, err
	}

	if !user.Data.IsAdmin {
		return []models.UserPostgres{}, errors.New("Forbidden")
	}

	postgresUsers := []models.UserPostgres{}
	err = db.DB.Model(&postgresUsers).Select()
	if err != nil {
		log.Printf("%v", err)
		return []models.UserPostgres{}, err
	}

	for i := 0; i < len(postgresUsers); i++ {
		postgresUsers[i].Data.Type = "users"
		postgresUsers[i].Data.Id = postgresUsers[i].Id
	}

	return postgresUsers, nil
}

func getUserUnauthorized(r *http.Request, id int64) (models.UserPostgres, error) {
	// Get the current signed in user details by Id
	postgresUser := models.UserPostgres{}
	err := db.DB.Model(&postgresUser).Where("id = ?", id).Select()
	if err != nil {
		log.Printf("%v", err)
		return models.UserPostgres{}, err
	}

	if postgresUser.Data.Email != "" {
		postgresUser.Data.Type = "users"
		postgresUser.Data.Id = postgresUser.Id
		return postgresUser, nil
	}

	return models.UserPostgres{}, errors.New("No user by this id")
}

func getUsersUnauthorized(r *http.Request) ([]models.UserPostgres, error) {
	// Get the current signed in user details by Id
	postgresUsers := []models.UserPostgres{}
	err := db.DB.Model(&postgresUsers).Select()
	if err != nil {
		log.Printf("%v", err)
		return []models.UserPostgres{}, err
	}

	for i := 0; i < len(postgresUsers); i++ {
		postgresUsers[i].Data.Type = "users"
		postgresUsers[i].Data.Id = postgresUsers[i].Id
	}

	return postgresUsers, nil
}

func filterUser(queryType, query string) (models.UserPostgres, error) {
	postgresUser := models.UserPostgres{}
	err := db.DB.Model(&postgresUser).Where("data->>'" + queryType + "' = '" + query + "'").Select()
	if err != nil {
		log.Printf("%v", err)
		return models.UserPostgres{}, err
	}

	if postgresUser.Data.Email != "" {
		postgresUser.Data.Type = "users"
		postgresUser.Data.Id = postgresUser.Id
		return postgresUser, nil
	}

	return models.UserPostgres{}, errors.New("No user by this " + queryType)
}

func filterUserConfirmed(queryType, query string) (models.UserPostgres, error) {
	postgresUsers := []models.UserPostgres{}
	err := db.DB.Model(&postgresUsers).Where("data->>'" + queryType + "' = '" + query + "'").Select()
	if err != nil {
		log.Printf("%v", err)
	}

	if len(postgresUsers) == 0 {
		return models.UserPostgres{}, errors.New("No user by this " + queryType)
	}

	if len(postgresUsers) > 1 {
		whichUserConfirmed := models.UserPostgres{}
		for i := 0; i < len(postgresUsers); i++ {
			postgresUsers[i].Data.Type = "users"
			postgresUsers[i].Data.Id = postgresUsers[i].Id

			if postgresUsers[i].Data.EmailConfirmed {
				whichUserConfirmed = postgresUsers[i]
			}
		}

		// If none of them have confirmed their email
		if whichUserConfirmed.Data.Email == "" {
			return postgresUsers[0], nil
		}

		return whichUserConfirmed, nil
	} else if len(postgresUsers) == 1 {
		user := postgresUsers[0]
		user.Data.Type = "users"
		user.Data.Id = user.Id
		return user, nil
	}

	return models.UserPostgres{}, errors.New("No user by this " + queryType)
}

/*
* Public methods
 */

/*
* Get methods
 */

func GetUsers(r *http.Request) ([]models.User, interface{}, int, int, error) {
	postgresUsers, err := getUsers(r)
	if err != nil {
		log.Printf("%v", err)
		return []models.User{}, nil, 0, 0, err
	}

	users := []models.User{}
	for i := 0; i < len(postgresUsers); i++ {
		postgresUsers[i].Data.Id = postgresUsers[i].Id
		users = append(users, postgresUsers[i].Data)
	}

	return users, nil, 0, 0, nil
}

func GetUser(r *http.Request, id string) (models.User, interface{}, error) {
	switch id {
	case "me":
		user, err := GetCurrentUser(r)
		if err != nil {
			log.Printf("%v", err)
			return models.User{}, nil, err
		}
		return user.Data, nil, err
	default:
		userId, err := utilities.StringIdToInt(id)
		if err != nil {
			log.Printf("%v", err)
			return models.User{}, nil, err
		}

		postgresUser, err := getUser(r, userId)
		if err != nil {
			log.Printf("%v", err)
			return models.User{}, nil, err
		}

		return postgresUser.Data, nil, nil
	}
}

func GetUsersUnauthorized(r *http.Request) ([]models.UserPostgres, error) {
	postgresUsers, err := getUsersUnauthorized(r)
	if err != nil {
		log.Printf("%v", err)
		return []models.UserPostgres{}, err
	}

	return postgresUsers, nil
}

func GetUserById(r *http.Request, id int64) (models.UserPostgres, interface{}, error) {
	postgresUser, err := getUser(r, id)
	if err != nil {
		log.Printf("%v", err)
		return models.UserPostgres{}, nil, err
	}

	return postgresUser, nil, nil
}

func GetUserByEmail(email string) (models.UserPostgres, error) {
	postgresUser, err := filterUser("email", email)
	if err != nil {
		log.Printf("%v", err)
		return models.UserPostgres{}, err
	}

	postgresUser.Data.Id = postgresUser.Id
	return postgresUser, nil
}

func GetUserByEmailForValidation(email string) (models.UserPostgres, error) {
	postgresUser, err := filterUserConfirmed("email", email)
	if err != nil {
		log.Printf("%v", err)
		return models.UserPostgres{}, err
	}

	return postgresUser, nil
}

func GetUserFromApiKey(apiKey string) (models.UserPostgres, error) {
	postgresUser, err := filterUser("apikey", apiKey)
	if err != nil {
		log.Printf("%v", err)
		return models.UserPostgres{}, err
	}

	return postgresUser, nil
}

func GetUserByConfirmationCode(confirmationCode string) (models.UserPostgres, error) {
	postgresUser, err := filterUser("confirmationcode", confirmationCode)
	if err != nil {
		log.Printf("%v", err)

		userBackup, errBackup := filterUser("confirmationcodebackup", confirmationCode)
		if errBackup != nil {
			log.Printf("%v", err)
			return models.UserPostgres{}, errBackup
		}

		return userBackup, nil
	}

	return postgresUser, nil
}

func GetUserByResetCode(resetCode string) (models.UserPostgres, error) {
	postgresUser, err := filterUser("resetpasswordcode", resetCode)
	if err != nil {
		log.Printf("%v", err)
		return models.UserPostgres{}, err
	}

	return postgresUser, nil
}

func GetCurrentUser(r *http.Request) (models.UserPostgres, error) {
	// Get the current user
	_, ok := gcontext.GetOk(r, "user")
	if !ok {
		return models.UserPostgres{}, errors.New("No user logged in")
	}

	user := gcontext.Get(r, "user").(models.UserPostgres)
	return user, nil
}

func GetUserByIdUnauthorized(r *http.Request, userId int64) (models.UserPostgres, error) {
	postgresUser, err := getUserUnauthorized(r, userId)
	if err != nil {
		log.Printf("%v", err)
		return models.UserPostgres{}, err
	}

	return postgresUser, nil
}

func AddUserToContext(r *http.Request, email string) {
	_, ok := gcontext.GetOk(r, "user")
	if !ok {
		user, _ := GetUserByEmail(email)
		gcontext.Set(r, "user", user)
		Update(r, &user)
	} else {
		user := gcontext.Get(r, "user").(models.UserPostgres)
		Update(r, &user)
	}
}

func AddPlanToUser(r *http.Request, id string) (models.User, interface{}, error) {
	postgresUser := models.UserPostgres{}
	err := errors.New("")

	switch id {
	case "me":
		postgresUser, err = GetCurrentUser(r)
		if err != nil {
			log.Printf("%v", err)
			return models.User{}, nil, err
		}
	default:
		userId, err := utilities.StringIdToInt(id)
		if err != nil {
			log.Printf("%v", err)
			return models.User{}, nil, err
		}
		postgresUser, err = getUser(r, userId)
		if err != nil {
			log.Printf("%v", err)
			return models.User{}, nil, err
		}
	}

	currentUser, err := GetCurrentUser(r)
	if err != nil {
		log.Printf("%v", err)
		return models.User{}, nil, err
	}

	if !currentUser.Data.IsAdmin {
		err = errors.New("Forbidden")
		log.Printf("%v", err)
		return models.User{}, nil, err
	}

	buf, _ := ioutil.ReadAll(r.Body)
	decoder := ffjson.NewDecoder()
	var userNewPlan models.UserNewPlan
	err = decoder.Decode(buf, &userNewPlan)
	if err != nil {
		log.Printf("%v", err)
		return models.User{}, nil, err
	}

	userBilling, err := GetUserBilling(r, postgresUser)
	if err != nil {
		return models.User{}, nil, err
	}

	if len(userBilling.Data.CardsOnFile) == 0 {
		return models.User{}, nil, errors.New("This user has no cards on file")
	}

	originalPlan := ""
	switch userNewPlan.Plan {
	case "bronze":
		originalPlan = "Personal"
	case "silver-1":
		originalPlan = "Freelancer"
	case "gold-1":
		originalPlan = "Business"
	}

	if userNewPlan.Duration != "monthly" && userNewPlan.Duration != "annually" {
		return models.User{}, nil, errors.New("Duration is invalid")
	}

	if userNewPlan.Plan == "" {
		return models.User{}, nil, errors.New("Plan is invalid")
	}

	if originalPlan == "" {
		return models.User{}, nil, errors.New("Original Plan is invalid")
	}

	err = billing.AddPlanToUser(r, postgresUser, &userBilling, userNewPlan.Plan, userNewPlan.Duration, userNewPlan.Coupon, originalPlan)
	if err != nil {
		log.Printf("%v", err)
		return models.User{}, nil, err
	}

	return postgresUser.Data, userBilling.Data, nil
}

func AddEmailToUser(r *http.Request, id string) (models.User, interface{}, error) {
	user := models.UserPostgres{}
	err := errors.New("")

	switch id {
	case "me":
		user, err = GetCurrentUser(r)
		if err != nil {
			log.Printf("%v", err)
			return models.User{}, nil, err
		}
	default:
		userId, err := utilities.StringIdToInt(id)
		if err != nil {
			log.Printf("%v", err)
			return models.User{}, nil, err
		}
		user, err = getUser(r, userId)
		if err != nil {
			log.Printf("%v", err)
			return models.User{}, nil, err
		}
	}

	currentUser, err := GetCurrentUser(r)
	if err != nil {
		log.Printf("%v", err)
		return models.User{}, nil, err
	}

	if !permissions.AccessToObject(user.Id, currentUser.Id) && !currentUser.Data.IsAdmin {
		err = errors.New("Forbidden")
		log.Printf("%v", err)
		return models.User{}, nil, err
	}

	// Only available when using SendGrid
	if user.Data.Gmail || user.Data.ExternalEmail {
		err = errors.New("Feature only works when using Sendgrid")
		log.Printf("%v", err)
		return models.User{}, nil, err
	}

	buf, _ := ioutil.ReadAll(r.Body)
	decoder := ffjson.NewDecoder()
	var userEmail models.UserEmail
	err = decoder.Decode(buf, &userEmail)
	if err != nil {
		log.Printf("%v", err)
		return models.User{}, nil, err
	}

	userEmail.Email = strings.ToLower(userEmail.Email)
	validEmail, err := mail.ParseAddress(userEmail.Email)
	if err != nil {
		log.Printf("%v", err)
		return models.User{}, nil, err
	}

	if user.Data.Email == validEmail.Address {
		err = errors.New("Can't add your default email as an extra email")
		log.Printf("%v", err)
		return models.User{}, nil, err
	}

	for i := 0; i < len(user.Data.Emails); i++ {
		if user.Data.Emails[i] == validEmail.Address {
			err = errors.New("Email already exists for the user")
			log.Printf("%v", err)
			return models.User{}, nil, err
		}
	}

	userEmailCode := models.UserEmailCode{}
	userEmailCode.InviteCode = utilities.RandToken()
	userEmailCode.Email = validEmail.Address
	userEmailCode.Create(r, currentUser)

	// Send Confirmation Email to this email address
	err = emails.AddEmailToUser(user.Data, validEmail.Address, userEmailCode.InviteCode)
	if err != nil {
		log.Printf("%v", err)
		return user.Data, nil, err
	}

	return user.Data, nil, nil
}

func RemoveEmailFromUser(r *http.Request, id string) (models.User, interface{}, error) {
	user := models.UserPostgres{}
	err := errors.New("")

	currentUser, err := GetCurrentUser(r)
	if err != nil {
		log.Printf("%v", err)
		return models.User{}, nil, err
	}

	switch id {
	case "me":
		user = currentUser
	default:
		userId, err := utilities.StringIdToInt(id)
		if err != nil {
			log.Printf("%v", err)
			return models.User{}, nil, err
		}
		user, err = getUser(r, userId)
		if err != nil {
			log.Printf("%v", err)
			return models.User{}, nil, err
		}
	}

	if !permissions.AccessToObject(user.Id, currentUser.Id) && !currentUser.Data.IsAdmin {
		err = errors.New("Forbidden")
		log.Printf("%v", err)
		return models.User{}, nil, err
	}

	buf, _ := ioutil.ReadAll(r.Body)
	decoder := ffjson.NewDecoder()
	var userEmail models.UserEmail
	err = decoder.Decode(buf, &userEmail)
	if err != nil {
		log.Printf("%v", err)
		return models.User{}, nil, err
	}

	userEmail.Email = strings.ToLower(userEmail.Email)
	validEmail, err := mail.ParseAddress(userEmail.Email)
	if err != nil {
		log.Printf("%v", err)
		return user.Data, nil, err
	}

	if user.Data.Email == validEmail.Address {
		err = errors.New("Can't remove your default email as an extra email")
		log.Printf("%v", err)
		return user.Data, nil, err
	}

	for i := 0; i < len(user.Data.Emails); i++ {
		if user.Data.Emails[i] == validEmail.Address {
			user.Data.Emails = append(user.Data.Emails[:i], user.Data.Emails[i+1:]...)
		}
	}

	SaveUser(r, &user)
	return user.Data, nil, nil
}

func GetUserDailyEmail(r *http.Request, user models.UserPostgres) int {
	return 0
}

func GetUserPlanDetails(r *http.Request, id string) (models.UserPlan, interface{}, error) {
	user := models.UserPostgres{}
	err := errors.New("")

	currentUser, err := GetCurrentUser(r)
	if err != nil {
		log.Printf("%v", err)
		return models.UserPlan{}, nil, err
	}

	switch id {
	case "me":
		user = currentUser
	default:
		userId, err := utilities.StringIdToInt(id)
		if err != nil {
			log.Printf("%v", err)
			return models.UserPlan{}, nil, err
		}
		user, err = getUser(r, userId)
		if err != nil {
			log.Printf("%v", err)
			return models.UserPlan{}, nil, err
		}
	}

	if !permissions.AccessToObject(user.Id, currentUser.Id) && !currentUser.Data.IsAdmin {
		err = errors.New("Forbidden")
		log.Printf("%v", err)
		return models.UserPlan{}, nil, err
	}

	userBilling, err := GetUserBilling(r, user)
	if err != nil {
		return models.UserPlan{}, nil, err
	}

	userPlan := models.UserPlan{}
	userPlanName := billing.BillingIdToPlanName(userBilling.Data.StripePlanId)
	userPlan.PlanName = userPlanName
	return userPlan, nil, nil
}

func ConfirmAddEmailToUser(r *http.Request, id string) (models.User, interface{}, error) {
	user := models.UserPostgres{}
	err := errors.New("")

	currentUser, err := GetCurrentUser(r)
	if err != nil {
		log.Printf("%v", err)
		return models.User{}, nil, err
	}

	switch id {
	case "me":
		user = currentUser
	default:
		userId, err := utilities.StringIdToInt(id)
		if err != nil {
			log.Printf("%v", err)
			return models.User{}, nil, err
		}
		user, err = getUser(r, userId)
		if err != nil {
			log.Printf("%v", err)
			return models.User{}, nil, err
		}
	}

	if !permissions.AccessToObject(user.Id, currentUser.Id) && !currentUser.Data.IsAdmin {
		err = errors.New("Forbidden")
		log.Printf("%v", err)
		return models.User{}, nil, err
	}

	if r.URL.Query().Get("code") != "" {
		userEmailCode := models.UserEmailCode{}
		err := db.DB.Model(&userEmailCode).Where("invite_code = ?", r.URL.Query().Get("code")).Select()
		if err != nil {
			log.Printf("%v", err)
			return models.User{}, nil, err
		}

		if userEmailCode.InviteCode != "" {
			if !permissions.AccessToObject(user.Id, userEmailCode.CreatedBy) {
				err = errors.New("Forbidden")
				log.Printf("%v", err)
				return models.User{}, nil, err
			}

			alreadyExists := false
			for i := 0; i < len(user.Data.Emails); i++ {
				if user.Data.Emails[i] == userEmailCode.Email {
					alreadyExists = true
				}
			}

			if !alreadyExists {
				user.Data.Emails = append(user.Data.Emails, userEmailCode.Email)
				SaveUser(r, &user)
			}

			return user.Data, nil, nil
		}

		return models.User{}, nil, errors.New("No code by the code you entered")
	}

	return models.User{}, nil, errors.New("No code present")
}

func FeedbackFromUser(r *http.Request, id string) (models.User, interface{}, error) {
	user := models.UserPostgres{}
	err := errors.New("")

	currentUser, err := GetCurrentUser(r)
	if err != nil {
		log.Printf("%v", err)
		return models.User{}, nil, err
	}

	switch id {
	case "me":
		user = currentUser
	default:
		userId, err := utilities.StringIdToInt(id)
		if err != nil {
			log.Printf("%v", err)
			return models.User{}, nil, err
		}
		user, err = getUser(r, userId)
		if err != nil {
			log.Printf("%v", err)
			return models.User{}, nil, err
		}
	}

	if !permissions.AccessToObject(user.Id, currentUser.Id) && !currentUser.Data.IsAdmin {
		err = errors.New("Forbidden")
		log.Printf("%v", err)
		return models.User{}, nil, err
	}

	buf, _ := ioutil.ReadAll(r.Body)
	decoder := ffjson.NewDecoder()
	var userFeedback models.UserFeedback
	err = decoder.Decode(buf, &userFeedback)
	if err != nil {
		log.Printf("%v", err)
		return models.User{}, nil, err
	}

	// Get user's billing profile and add reasons there
	userBilling, err := GetUserBilling(r, currentUser)
	userBilling.Data.ReasonNotPurchase = userFeedback.ReasonNotPurchase
	userBilling.Data.FeedbackAfterTrial = userFeedback.FeedbackAfterTrial
	userBilling.Save()

	// Set the trial feedback to true - since they gave us feedback now
	user.Data.TrialFeedback = true
	user.Save()

	// sync.ResourceSync(r, user.Id, "User", "create")
	return user.Data, nil, nil
}

/*
* Update methods
 */

func SaveUser(r *http.Request, u *models.UserPostgres) (*models.UserPostgres, error) {
	u.Save()
	// sync.ResourceSync(r, u.Id, "User", "create")
	return u, nil
}

func Update(r *http.Request, u *models.UserPostgres) (*models.UserPostgres, error) {
	if len(u.Data.Employers) == 0 {
		// CreateAgencyFromUser( r, u)
	}

	billing, err := GetUserBilling(r, *u)
	if err != nil {
		return u, err
	}

	if billing.Data.Expires.Before(time.Now()) {
		if billing.Data.IsOnTrial {
			u.Data.IsActive = false
			u.Save()

			billing.Data.IsOnTrial = false
			billing.Save()
		} else {
			if billing.Data.IsCancel {
				u.Data.IsActive = false
				u.Save()
			} else {
				if billing.Data.StripePlanId != "free" {
					// If they haven't canceled then we can add a month until they do.
					// More sophisticated to add the amount depending on what
					// plan they were on.
					addAMonth := billing.Data.Expires.AddDate(0, 1, 0)
					billing.Data.Expires = addAMonth
					billing.Save()

					// Keep the user active
					u.Data.IsActive = true
					u.Save()
				}
			}
		}
	}

	return u, nil
}

func UpdateUser(r *http.Request, id string) (models.User, interface{}, error) {
	user := models.UserPostgres{}
	err := errors.New("")

	currentUser, err := GetCurrentUser(r)
	if err != nil {
		log.Printf("%v", err)
		return models.User{}, nil, err
	}

	switch id {
	case "me":
		user = currentUser
	default:
		userId, err := utilities.StringIdToInt(id)
		if err != nil {
			log.Printf("%v", err)
			return models.User{}, nil, err
		}
		user, err = getUser(r, userId)
		if err != nil {
			log.Printf("%v", err)
			return models.User{}, nil, err
		}
	}

	if !permissions.AccessToObject(user.Id, currentUser.Id) && !currentUser.Data.IsAdmin {
		err = errors.New("Forbidden")
		log.Printf("%v", err)
		return models.User{}, nil, err
	}

	buf, _ := ioutil.ReadAll(r.Body)
	decoder := ffjson.NewDecoder()
	var updatedUser models.User
	err = decoder.Decode(buf, &updatedUser)
	if err != nil {
		log.Printf("%v", err)
		return models.User{}, nil, err
	}

	utilities.UpdateIfNotBlank(&user.Data.FirstName, updatedUser.FirstName)
	utilities.UpdateIfNotBlank(&user.Data.LastName, updatedUser.LastName)
	utilities.UpdateIfNotBlank(&user.Data.EmailSignature, updatedUser.EmailSignature)

	// If new user wants to get daily emails
	if updatedUser.GetDailyEmails == true {
		user.Data.GetDailyEmails = true
	}

	// If this person doesn't want to get daily emails anymore
	if user.Data.GetDailyEmails == true && updatedUser.GetDailyEmails == false {
		user.Data.GetDailyEmails = false
	}

	if user.Data.SMTPValid {
		// If new user wants to get daily emails
		if updatedUser.ExternalEmail == true {
			user.Data.ExternalEmail = true
		}

		// If this person doesn't want to get daily emails anymore
		if user.Data.ExternalEmail == true && updatedUser.ExternalEmail == false {
			user.Data.ExternalEmail = false
		}
	}

	if len(updatedUser.Employers) > 0 {
		user.Data.Employers = updatedUser.Employers
	}

	if len(updatedUser.EmailSignatures) > 0 {
		user.Data.EmailSignatures = updatedUser.EmailSignatures
	}

	// Special case when you want to remove all the email signatures
	if len(user.Data.EmailSignatures) > 0 && len(updatedUser.EmailSignatures) == 0 {
		user.Data.EmailSignatures = updatedUser.EmailSignatures
	}

	user.Save()
	// sync.ResourceSync(r, user.Id, "User", "create")
	return user.Data, nil, nil

}

/*
* Action methods
 */

func BanUser(r *http.Request, id string) (models.User, interface{}, error) {
	user := models.UserPostgres{}
	err := errors.New("")

	currentUser, err := GetCurrentUser(r)
	if err != nil {
		log.Printf("%v", err)
		return models.User{}, nil, err
	}

	switch id {
	case "me":
		user = currentUser
	default:
		userId, err := utilities.StringIdToInt(id)
		if err != nil {
			log.Printf("%v", err)
			return models.User{}, nil, err
		}
		user, err = getUser(r, userId)
		if err != nil {
			log.Printf("%v", err)
			return models.User{}, nil, err
		}
	}

	if !permissions.AccessToObject(user.Id, currentUser.Id) && !currentUser.Data.IsAdmin {
		err = errors.New("Forbidden")
		log.Printf("%v", err)
		return models.User{}, nil, err
	}

	user.Data.IsActive = false
	user.Data.IsBanned = true
	SaveUser(r, &user)
	return user.Data, nil, nil
}

func GetAndRefreshLiveToken(r *http.Request, id string) (models.UserLiveToken, interface{}, error) {
	user := models.UserPostgres{}
	err := errors.New("")

	currentUser, err := GetCurrentUser(r)
	if err != nil {
		log.Printf("%v", err)
		return models.UserLiveToken{}, nil, err
	}

	switch id {
	case "me":
		user = currentUser
	default:
		userId, err := utilities.StringIdToInt(id)
		if err != nil {
			log.Printf("%v", err)
			return models.UserLiveToken{}, nil, err
		}
		user, err = getUser(r, userId)
		if err != nil {
			log.Printf("%v", err)
			return models.UserLiveToken{}, nil, err
		}
	}

	if !permissions.AccessToObject(user.Id, currentUser.Id) && !currentUser.Data.IsAdmin {
		err = errors.New("Forbidden")
		log.Printf("%v", err)
		return models.UserLiveToken{}, nil, err
	}

	token := models.UserLiveToken{}
	token.Token = user.Data.LiveAccessToken
	token.Expires = user.Data.LiveAccessTokenExpire
	return token, nil, nil
}

func ValidateUserPassword(r *http.Request, email string, password string) (models.UserPostgres, bool, error) {
	user, err := GetUserByEmailForValidation(email)
	if err == nil {
		err = utilities.ValidatePassword(user.Data.Password, password)
		if err != nil {
			log.Printf("%v", err)
			return user, false, nil
		}

		return user, true, nil
	}

	return models.UserPostgres{}, false, errors.New("User does not exist")

}

func SetUser(r *http.Request, userId int64) (models.UserPostgres, error) {
	user, err := getUserUnauthorized(r, userId)
	if err != nil {
		log.Printf("%v", err)
	}

	gcontext.Set(r, "user", user)
	return user, nil

}

func UpdateUserEmail(r *http.Request, id string) (models.User, interface{}, error) {
	user := models.UserPostgres{}
	err := errors.New("")

	currentUser, err := GetCurrentUser(r)
	if err != nil {
		log.Printf("%v", err)
		return models.User{}, nil, err
	}

	switch id {
	case "me":
		user = currentUser
	default:
		userId, err := utilities.StringIdToInt(id)
		if err != nil {
			log.Printf("%v", err)
			return models.User{}, nil, err
		}
		user, err = getUser(r, userId)
		if err != nil {
			log.Printf("%v", err)
			return models.User{}, nil, err
		}
	}

	if !permissions.AccessToObject(user.Id, currentUser.Id) && !currentUser.Data.IsAdmin {
		err = errors.New("Forbidden")
		log.Printf("%v", err)
		return models.User{}, nil, err
	}

	buf, _ := ioutil.ReadAll(r.Body)
	decoder := ffjson.NewDecoder()
	var updatedUser models.User
	err = decoder.Decode(buf, &updatedUser)
	if err != nil {
		log.Printf("%v", err)
		return models.User{}, nil, err
	}

	// If new user wants to get daily emails
	if updatedUser.Email != "" {
		user.Data.Email = updatedUser.Email
	}

	user.Save()
	// sync.ResourceSync(r, user.Id, "User", "create")
	return user.Data, nil, nil
}
