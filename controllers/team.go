package controllers

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/pquerna/ffjson/ffjson"

	"github.com/news-ai/api-v1/db"
	"github.com/news-ai/api-v1/models"

	"github.com/news-ai/web/utilities"
)

/*
* Private methods
 */

/*
* Get methods
 */

func getTeam(id int64) (models.Team, error) {
	if id == 0 {
		return models.Team{}, errors.New("datastore: no such entity")
	}

	// Get the team details by id
	var team models.Team
	err := db.DB.Model(&team).Where("id = ?", id).Select()
	if err != nil {
		log.Printf("%v", err)
		return models.Team{}, err
	}

	if !team.Created.IsZero() {
		team.Type = "teams"
		return team, nil
	}

	return models.Team{}, errors.New("No team by this id")
}

/*
* Public methods
 */

/*
* Get methods
 */

func GetTeams(r *http.Request) ([]models.Team, interface{}, int, int, error) {
	// Now if user is not querying then check
	user, err := GetCurrentUser(r)
	if err != nil {
		log.Printf("%v", err)
		return []models.Team{}, nil, 0, 0, err
	}

	if !user.Data.IsAdmin {
		return []models.Team{}, nil, 0, 0, errors.New("Forbidden")
	}

	teams := []models.Team{}
	err = db.DB.Model(&teams).Where("created_by = ?", user.Id).Select()
	if err != nil {
		log.Printf("%v", err)
		return []models.Team{}, nil, 0, 0, err
	}

	for i := 0; i < len(teams); i++ {
		teams[i].Type = "teams"
	}

	return teams, nil, len(teams), 0, nil
}

func GetTeam(id string) (models.Team, interface{}, error) {
	// Get the details of the current team
	currentId, err := utilities.StringIdToInt(id)
	if err != nil {
		log.Printf("%v", err)
		return models.Team{}, nil, err
	}

	team, err := getTeam(currentId)
	if err != nil {
		log.Printf("%v", err)
		return models.Team{}, nil, err
	}
	return team, nil, nil
}

/*
* Create methods
 */

func CreateTeam(r *http.Request) ([]models.Team, interface{}, error) {
	buf, _ := ioutil.ReadAll(r.Body)

	currentUser, err := GetCurrentUser(r)
	if err != nil {
		log.Printf("%v", err)
		return []models.Team{}, nil, err
	}

	if !currentUser.Data.IsAdmin {
		return []models.Team{}, nil, errors.New("Forbidden")
	}

	decoder := ffjson.NewDecoder()
	var team models.Team
	err = decoder.Decode(buf, &team)
	if err != nil {
		log.Printf("%v", err)
		return []models.Team{}, nil, err
	}

	if len(team.Members) > team.MaxMembers {
		return []models.Team{}, nil, errors.New("The number of members is greater than the allowed number of members")
	}

	// Create team
	_, err = team.Create(r, currentUser)
	if err != nil {
		log.Printf("%v", err)
		return []models.Team{}, nil, err
	}

	// Add team Id to team members
	// Validate member accounts
	confirmMembers := []int64{}
	for i := 0; i < len(team.Members); i++ {
		user, err := getUser(r, team.Members[i])
		if err == nil && user.Data.TeamId == 0 {
			confirmMembers = append(confirmMembers, user.Id)
			user.Data.TeamId = team.Id
			user.Save()
		}
	}

	// Validate admin accounts
	confirmAdmins := []int64{}
	for i := 0; i < len(team.Admins); i++ {
		user, err := getUser(r, team.Admins[i])
		if err == nil {
			confirmAdmins = append(confirmAdmins, user.Id)
		}
	}

	team.Members = confirmMembers
	team.Admins = confirmAdmins
	team.Save()

	return []models.Team{team}, nil, nil
}
