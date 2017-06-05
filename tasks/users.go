package tasks

import (
	"net/http"
	"time"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"

	"github.com/news-ai/api/controllers"

	"github.com/news-ai/tabulae/sync"

	"github.com/news-ai/web/errors"
)

func MakeUsersInactive(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	users, err := controllers.GetUsersUnauthorized(c, r)
	if err != nil {
		log.Errorf(c, "%v", err)
		errors.ReturnError(w, http.StatusInternalServerError, "Could not get users", err.Error())
		return
	}

	for i := 0; i < len(users); i++ {
		billing, err := controllers.GetUserBilling(c, r, users[i])
		if err != nil {
			log.Errorf(c, "%v", users[i])
			log.Errorf(c, "%v", err)
			continue
		}

		// For now only consider when they are on trial
		if billing.IsOnTrial {
			if billing.Expires.Before(time.Now()) {
				users[i].IsActive = false
				users[i].Save(c)
				sync.ResourceSync(r, users[i].Id, "User", "create")

				billing.IsOnTrial = false
				billing.IsCancel = true
				billing.Save(c)
			}
		}
	}
}
