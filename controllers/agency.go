package controllers

import (
	"errors"
	"log"
	"net/http"
	"time"

	// gcontext "github.com/gorilla/context"

	"github.com/news-ai/web/utilities"

	"github.com/news-ai/api-v1/db"
	"github.com/news-ai/api-v1/models"
	// "github.com/news-ai/tabulae-v1/search"
)

/*
* Private methods
 */

/*
* Get methods
 */

func getAgency(id int64) (models.Agency, error) {
	agency := models.Agency{}
	err := db.DB.Model(&agency).Where("id = ?", id).Select()
	if err != nil {
		log.Printf("%v", err)
		return models.Agency{}, err
	}

	if !agency.Created.IsZero() {
		agency.Type = "agencies"
		return agency, nil
	}

	return models.Agency{}, errors.New("No agency by this id")
}

/*
* Filter methods
 */

func filterAgency(queryType, query string) (models.Agency, error) {
	agency := models.Agency{}
	err := db.DB.Model(&agency).Where(queryType+" = ?", query).Select()
	if err != nil {
		log.Printf("%v", err)
		return models.Agency{}, err
	}

	if agency.Email != "" {
		agency.Type = "agencies"
		return agency, nil
	}

	return models.Agency{}, errors.New("No agency by this " + queryType)
}

/*
* Public methods
 */

/*
* Get methods
 */

// Gets every single agency
func GetAgencies(r *http.Request) ([]models.Agency, interface{}, int, int, error) {
	// If user is querying then it is not denied by the server
	// queryField := gcontext.Get(r, "q").(string)
	// if queryField != "" {
	// 	agencies, total, err := search.SearchAgency(c, r, queryField)
	// 	if err != nil {
	// 		return []models.Agency{}, nil, 0, 0, err
	// 	}
	// 	return agencies, nil, len(agencies), total, nil
	// }

	// Now if user is not querying then check
	// user, err := GetCurrentUser(r)
	// if err != nil {
	// 	log.Printf("%v", err)
	// 	return []models.Agency{}, nil, 0, 0, err
	// }

	// if !user.IsAdmin {
	// 	return []models.Agency{}, nil, 0, 0, errors.New("Forbidden")
	// }

	// query := datastore.NewQuery("Agency")
	// query = ConstructQuery(query, r)

	// ks, err := query.KeysOnly().GetAll(c, nil)
	// if err != nil {
	// 	log.Printf("%v", err)
	// 	return []models.Agency{}, nil, 0, 0, err
	// }

	var agencies []models.Agency
	// agencies = make([]models.Agency, len(ks))
	// err = nds.GetMulti(c, ks, agencies)
	// if err != nil {
	// 	log.Printf("%v", err)
	// 	return agencies, nil, 0, 0, err
	// }

	// for i := 0; i < len(agencies); i++ {
	// 	agencies[i].Format(ks[i], "agencies")
	// }

	return agencies, nil, len(agencies), 0, nil
}

func GetAgency(id string) (models.Agency, interface{}, error) {
	// Get the details of the current agency
	currentId, err := utilities.StringIdToInt(id)
	if err != nil {
		log.Printf("%v", err)
		return models.Agency{}, nil, err
	}

	agency, err := getAgency(currentId)
	if err != nil {
		log.Printf("%v", err)
		return models.Agency{}, nil, err
	}

	return agency, nil, nil
}

/*
* Create methods
 */

func CreateAgencyFromUser(r *http.Request, u *models.UserPostgres) (models.Agency, error) {
	agencyEmail, err := utilities.ExtractEmailExtension(u.Data.Email)
	if err != nil {
		log.Printf("%v", err)
		return models.Agency{}, err
	}

	agency, err := FilterAgencyByEmail(agencyEmail)
	if err != nil {
		agency = models.Agency{}
		agency.Name, err = utilities.ExtractNameFromEmail(agencyEmail)
		agency.Email = agencyEmail
		agency.Created = time.Now()

		// The person who signs up for the agency at the beginning
		// becomes the defacto administrator until we change.
		currentUser, err := GetCurrentUser(r)
		if err != nil {
			log.Printf("%v", err)
			return agency, err
		}

		agency.Create(r, currentUser)
	}

	u.Data.Employers = append(u.Data.Employers, agency.Id)
	u.Save()
	return agency, nil
}

/*
* Filter methods
 */

func FilterAgencyByEmail(email string) (models.Agency, error) {
	// Get the id of the current agency
	agency, err := filterAgency("Email", email)
	if err != nil {
		log.Printf("%v", err)
		return models.Agency{}, err
	}

	return agency, nil
}
