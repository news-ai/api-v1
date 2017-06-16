package search

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"golang.org/x/net/context"

	gcontext "github.com/gorilla/context"

	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"

	// pitchModels "github.com/news-ai/pitch/models"

	elastic "github.com/news-ai/elastic-appengine"
)

var (
	elasticContactDatabase *elastic.Elastic
	elasticMediaDatabase   *elastic.Elastic
)

type EnhanceResponse struct {
	Data interface{} `json:"data"`
}

type EnhanceFullContactProfileResponse struct {
	Data struct {
		Status        int `json:"status"`
		Organizations []struct {
			StartDate string `json:"startDate,omitempty"`
			EndDate   string `json:"endDate,omitempty"`
			Name      string `json:"name,omitempty"`
			Title     string `json:"title"`
		} `json:"organizations"`
		DigitalFootprint struct {
			Topics []struct {
				Value    string `json:"value"`
				Provider string `json:"provider"`
			} `json:"topics"`
			Scores []struct {
				Type     string `json:"type"`
				Value    int    `json:"value"`
				Provider string `json:"provider"`
			} `json:"scores"`
		} `json:"digitalFootprint"`
		SocialProfiles []struct {
			Username  string `json:"username,omitempty"`
			Bio       string `json:"bio,omitempty"`
			TypeID    string `json:"typeId"`
			URL       string `json:"url"`
			TypeName  string `json:"typeName"`
			Type      string `json:"type"`
			Followers int    `json:"followers,omitempty"`
			ID        string `json:"id,omitempty"`
			Following int    `json:"following,omitempty"`
		} `json:"socialProfiles"`
		Demographics struct {
			LocationDeduced struct {
				City struct {
					Name string `json:"name"`
				} `json:"city"`
				Country struct {
					Code    string `json:"code"`
					Name    string `json:"name"`
					Deduced bool   `json:"deduced"`
				} `json:"country"`
				DeducedLocation string `json:"deducedLocation"`
				State           struct {
					Code string `json:"code"`
					Name string `json:"name"`
				} `json:"state"`
				NormalizedLocation string  `json:"normalizedLocation"`
				Likelihood         float64 `json:"likelihood"`
				Continent          struct {
					Name    string `json:"name"`
					Deduced bool   `json:"deduced"`
				} `json:"continent"`
			} `json:"locationDeduced"`
			Gender          string `json:"gender"`
			LocationGeneral string `json:"locationGeneral"`
		} `json:"demographics"`
		Photos []struct {
			URL       string `json:"url"`
			TypeID    string `json:"typeId"`
			IsPrimary bool   `json:"isPrimary,omitempty"`
			Type      string `json:"type"`
			TypeName  string `json:"typeName"`
		} `json:"photos"`
		RequestID   string `json:"requestId"`
		ContactInfo struct {
			GivenName  string `json:"givenName"`
			FullName   string `json:"fullName"`
			FamilyName string `json:"familyName"`
			Websites   []struct {
				URL string `json:"url"`
			} `json:"websites"`
		} `json:"contactInfo"`
		Likelihood float64 `json:"likelihood"`
	} `json:"data"`
}

type EnhanceFullContactCompanyResponse struct {
	Data struct {
		Status    int    `json:"status"`
		RequestID string `json:"requestId"`
		Category  []struct {
			Name string `json:"name"`
			Code string `json:"code"`
		} `json:"category"`
		Logo           string `json:"logo"`
		Website        string `json:"website"`
		LanguageLocale string `json:"languageLocale"`
		OnlineSince    string `json:"onlineSince"`
		Organization   struct {
			Name            string `json:"name"`
			ApproxEmployees int    `json:"approxEmployees"`
			Founded         string `json:"founded"`
			ContactInfo     struct {
				EmailAddresses []struct {
					Value string `json:"value"`
					Label string `json:"label"`
				} `json:"emailAddresses"`
				PhoneNumbers []struct {
					Number string `json:"number"`
					Label  string `json:"label"`
				} `json:"phoneNumbers"`
				Addresses []struct {
					AddressLine1 string `json:"addressLine1"`
					Locality     string `json:"locality"`
					Region       struct {
						Name string `json:"name"`
						Code string `json:"code"`
					} `json:"region"`
					Country struct {
						Name string `json:"name"`
						Code string `json:"code"`
					} `json:"country"`
					PostalCode string `json:"postalCode"`
					Label      string `json:"label"`
				} `json:"addresses"`
			} `json:"contactInfo"`
			Links []struct {
				URL   string `json:"url"`
				Label string `json:"label"`
			} `json:"links"`
			Images []struct {
				URL   string `json:"url"`
				Label string `json:"label"`
			} `json:"images"`
			Keywords []string `json:"keywords"`
		} `json:"organization"`
		SocialProfiles []struct {
			TypeID    string `json:"typeId"`
			TypeName  string `json:"typeName"`
			URL       string `json:"url"`
			Bio       string `json:"bio,omitempty"`
			Followers int    `json:"followers,omitempty"`
			Following int    `json:"following,omitempty"`
			Username  string `json:"username,omitempty"`
			ID        string `json:"id,omitempty"`
		} `json:"socialProfiles"`
	} `json:"data"`
}

type DatabaseResponse struct {
	Email string      `json:"email"`
	Data  interface{} `json:"data"`
}

func searchContactInMediaDatabase(c context.Context, elasticQuery interface{}) (interface{}, error) {
	hits, err := elasticMediaDatabase.QueryStruct(c, elasticQuery)
	if err != nil {
		log.Errorf(c, "%v", err)
		return nil, err
	}

	profileHits := hits.Hits

	if len(profileHits) == 0 {
		log.Infof(c, "%v", profileHits)
		return nil, errors.New("No Media Database profile for this email")
	}

	var interfaceSlice = make([]interface{}, len(profileHits))

	for i := 0; i < len(profileHits); i++ {
		interfaceSlice[i] = profileHits[i].Source.Data
	}

	return interfaceSlice, nil
}

func searchESContactsDatabase(c context.Context, elasticQuery elastic.ElasticQuery) (interface{}, int, int, error) {
	hits, err := elasticContactDatabase.QueryStruct(c, elasticQuery)
	if err != nil {
		log.Errorf(c, "%v", err)
		return nil, 0, 0, err
	}

	contactHits := hits.Hits
	var contacts []interface{}
	for i := 0; i < len(contactHits); i++ {
		rawContact := contactHits[i].Source.Data
		contactData := DatabaseResponse{
			Email: contactHits[i].ID,
			Data:  rawContact,
		}
		contacts = append(contacts, contactData)
	}

	return contacts, len(contactHits), hits.Total, nil
}

func SearchCompanyDatabase(c context.Context, r *http.Request, url string) (EnhanceFullContactCompanyResponse, error) {
	contextWithTimeout, _ := context.WithTimeout(c, time.Second*15)
	client := urlfetch.Client(contextWithTimeout)
	getUrl := "https://enhance.newsai.org/company/" + url

	req, _ := http.NewRequest("GET", getUrl, nil)

	resp, err := client.Do(req)
	if err != nil {
		log.Errorf(c, "%v", err)
		return EnhanceFullContactCompanyResponse{}, err
	}

	var enhanceResponse EnhanceFullContactCompanyResponse
	err = json.NewDecoder(resp.Body).Decode(&enhanceResponse)
	if err != nil {
		log.Errorf(c, "%v", err)
		return EnhanceFullContactCompanyResponse{}, err
	}

	return enhanceResponse, nil
}

func SearchContactDatabase(c context.Context, r *http.Request, email string) (EnhanceFullContactProfileResponse, error) {
	contextWithTimeout, _ := context.WithTimeout(c, time.Second*15)
	client := urlfetch.Client(contextWithTimeout)
	getUrl := "https://enhance.newsai.org/fullcontact/" + email

	req, _ := http.NewRequest("GET", getUrl, nil)

	resp, err := client.Do(req)
	if err != nil {
		log.Errorf(c, "%v", err)
		return EnhanceFullContactProfileResponse{}, err
	}

	var enhanceResponse EnhanceFullContactProfileResponse
	err = json.NewDecoder(resp.Body).Decode(&enhanceResponse)
	if err != nil {
		log.Errorf(c, "%v", err)
		return EnhanceFullContactProfileResponse{}, err
	}

	return enhanceResponse, nil
}

func SearchContactInMediaDatabase(c context.Context, r *http.Request, email string) (interface{}, error) {
	if email == "" {
		return nil, nil
	}

	elasticQuery := ElasticMGetQuery{}
	elasticQuery.Ids = append(elasticQuery.Ids, email)
	return searchContactInMediaDatabase(c, elasticQuery)
}

func SearchESContactsDatabase(c context.Context, r *http.Request) (interface{}, int, int, error) {
	offset := gcontext.Get(r, "offset").(int)
	limit := gcontext.Get(r, "limit").(int)

	elasticQuery := elastic.ElasticQuery{}
	elasticQuery.Size = limit
	elasticQuery.From = offset

	return searchESContactsDatabase(c, elasticQuery)
}
