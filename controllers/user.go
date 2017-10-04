package controllers

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"

	gcontext "github.com/gorilla/context"
	"github.com/pquerna/ffjson/ffjson"

	"github.com/news-ai/api-v1/billing"
	"github.com/news-ai/api-v1/db"
	"github.com/news-ai/api-v1/models"

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

func GetUserByApiKey(apiKey string) (models.UserPostgres, error) {
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

func Update(r *http.Request, u *models.UserPostgres) (*models.UserPostgres, error) {
	return u, nil
}
