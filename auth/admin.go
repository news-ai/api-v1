package auth

import (
	"errors"
	"net/http"
	"text/template"

	"google.golang.org/appengine"

	"github.com/news-ai/tabulae/controllers"

	nError "github.com/news-ai/web/errors"
)

func AdminPageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := appengine.NewContext(r)
		user, _ := controllers.GetCurrentUser(c, r)

		if !user.IsAdmin {
			err := errors.New("Forbidden")
			w.Header().Set("Content-Type", "application/json")
			nError.ReturnError(w, http.StatusForbidden, "Forbidden", err.Error())
			return
		}

		numberEmailsScheduled, _ := controllers.GetNumberOfScheduledEmails(c, r)
		getNumberOfEmailsCreatedToday, _ := controllers.GetNumberOfEmailsCreatedToday(c, r)
		getNumberOfEmailsCreatedMonth, _ := controllers.GetNumberOfEmailsCreatedMonth(c, r)

		data := map[string]interface{}{
			"numberEmailsScheduled":         numberEmailsScheduled,
			"getNumberOfEmailsCreatedToday": getNumberOfEmailsCreatedToday,
			"getNumberOfEmailsCreatedMonth": getNumberOfEmailsCreatedMonth,
		}

		t := template.New("emails.html")
		t, _ = t.ParseFiles("admin/emails.html")
		t.Execute(w, data)
		return
	}
}
