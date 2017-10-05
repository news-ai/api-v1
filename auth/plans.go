package auth

import (
	"log"
	"net/http"
	"net/url"
	"strings"
	"text/template"
	"time"

	apiControllers "github.com/news-ai/api-v1/controllers"

	"github.com/news-ai/api-v1/billing"

	"github.com/gorilla/csrf"
	"github.com/pquerna/ffjson/ffjson"

	nError "github.com/news-ai/web/errors"
)

func TrialPlanPageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := apiControllers.GetCurrentUser(r)

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

		if !user.Data.IsActive {
			userBilling, err := apiControllers.GetUserBilling(r, user)

			// If the user has a user billing
			if err == nil {
				if userBilling.Data.HasTrial && !userBilling.Data.Expires.IsZero() {
					// If the user has already had a trial and has expired
					// This means: If userBilling Expire date is before the current time
					if userBilling.Data.Expires.Before(time.Now()) {
						http.Redirect(w, r, "/api/billing", 302)
						return
					}

					// If the user has already had a trial but it has not expired
					if userBilling.Data.Expires.After(time.Now()) {
						http.Redirect(w, r, "https://tabulae.newsai.co/", 302)
						return
					}
				}
			}

			billingId, err := billing.AddFreeTrialToUser(r, user, "free")
			user.Data.IsActive = true
			user.Data.BillingId = billingId
			user.Save()

			// If there was an error creating this person's trial
			if err != nil {
				log.Printf("%v", err)
				http.Redirect(w, r, "/api/billing/plans/trial", 302)
				return
			}

			// If they have a coupon they want to use (to expire later)
			if user.Data.PromoCode != "" {
				if user.Data.PromoCode == "PRCOUTURE" || user.Data.PromoCode == "GOPUBLIX" {
					userBilling, err := apiControllers.GetUserBilling(r, user)
					if err == nil {
						userBilling.Data.Expires = userBilling.Data.Expires.AddDate(0, 3, 0)
						userBilling.Save()
					} else {
						log.Printf("%v", err)
					}
				}
			}

			// If not then their is now probably successful so we redirect them back
			returnURL := "https://tabulae.newsai.co/"
			session, _ := store.Get(r, "sess")
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
			// If the user is active then they don't need to start a free trial
			http.Redirect(w, r, "https://tabulae.newsai.co/", 302)
			return
		}
	}
}

func CancelPlanPageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := apiControllers.GetCurrentUser(r)

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

		userBilling, err := apiControllers.GetUserBilling(r, user)

		// If the user has a billing profile
		if err == nil {
			switch userBilling.Data.StripePlanId {
			case "personal":
				userBilling.Data.StripePlanId = "Personal"
			case "consultant":
				userBilling.Data.StripePlanId = "Consultant"
			case "business":
				userBilling.Data.StripePlanId = "Business"
			case "growing":
				userBilling.Data.StripePlanId = "Growing Business"
			}

			userNotActiveNonTrialPlan := true
			if user.Data.IsActive && !userBilling.Data.IsOnTrial {
				userNotActiveNonTrialPlan = false
			}

			data := map[string]interface{}{
				"userNotActiveNonTrialPlan": userNotActiveNonTrialPlan,
				"currentUserPlan":           userBilling.Data.StripePlanId,
				"userEmail":                 user.Data.Email,
				csrf.TemplateTag:            csrf.TemplateField(r),
			}

			t := template.New("cancel.html")
			t, _ = t.ParseFiles("billing/cancel.html")
			t.Execute(w, data)
		} else {
			// If the user does not have billing profile that means that they
			// have not started their trial yet.
			http.Redirect(w, r, "/api/billing/plans/trial", 302)
			return
		}
	}
}

func CancelPlanHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// To check if there is a user logged in
		user, err := apiControllers.GetCurrentUser(r)

		if r.URL.Query().Get("next") != "" {
			session, _ := store.Get(r, "sess")
			session.Values["next"] = r.URL.Query().Get("next")
			session.Save(r, w)

			// If there is a next and the user has not been logged in
			if err != nil {
				log.Printf("%v", err)
				http.Redirect(w, r, r.URL.Query().Get("next"), 302)
				return
			}
		}

		// If there is no next and the user is not logged in
		if err != nil {
			log.Printf("%v", err)
			http.Redirect(w, r, "https://tabulae.newsai.co/", 302)
			return
		}

		userBilling, err := apiControllers.GetUserBilling(r, user)

		// If the user has a billing profile
		if err == nil {
			plan := ""
			switch userBilling.Data.StripePlanId {
			case "personal":
				plan = "Personal"
			case "consultant":
				plan = "Consultant"
			case "business":
				plan = "Business"
			case "growing":
				plan = "Growing Business"
			}

			data := map[string]interface{}{
				"plan":           plan,
				"userEmail":      user.Data.Email,
				csrf.TemplateTag: csrf.TemplateField(r),
			}

			t := template.New("cancelled.html")
			t, _ = t.ParseFiles("billing/confirmation.html")
			t.Execute(w, data)
		} else {
			// If the user does not have billing profile that means that they
			// have not started their trial yet.
			http.Redirect(w, r, "/api/billing/plans/trial", 302)
			return
		}
	}
}

func ChoosePlanPageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := apiControllers.GetCurrentUser(r)

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

		userBilling, err := apiControllers.GetUserBilling(r, user)

		// If the user has a billing profile
		if err == nil {
			userBilling.Data.StripePlanId = billing.BillingIdToPlanName(userBilling.Data.StripePlanId)

			userNotActiveNonTrialPlan := true
			if user.Data.IsActive && !userBilling.Data.IsOnTrial {
				userNotActiveNonTrialPlan = false
			}

			data := map[string]interface{}{
				"userNotActiveNonTrialPlan": userNotActiveNonTrialPlan,
				"currentUserPlan":           userBilling.Data.StripePlanId,
				"userEmail":                 user.Data.Email,
			}

			t := template.New("plans.html")
			t, _ = t.ParseFiles("billing/plans.html")
			t.Execute(w, data)
		} else {
			// If the user does not have billing profile that means that they
			// have not started their trial yet.
			http.Redirect(w, r, "/api/billing/plans/trial", 302)
			return
		}
	}
}

func ChooseSwitchPlanHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		plan := r.FormValue("plan")
		duration := r.FormValue("duration")

		// To check if there is a user logged in
		user, err := apiControllers.GetCurrentUser(r)

		if r.URL.Query().Get("next") != "" {
			session, _ := store.Get(r, "sess")
			session.Values["next"] = r.URL.Query().Get("next")
			session.Save(r, w)

			// If there is a next and the user has not been logged in
			if err != nil {
				log.Printf("%v", err)
				http.Redirect(w, r, r.URL.Query().Get("next"), 302)
				return
			}
		}

		// If there is no next and the user is not logged in
		if err != nil {
			log.Printf("%v", err)
			http.Redirect(w, r, "https://tabulae.newsai.co/", 302)
			return
		}

		userBilling, err := apiControllers.GetUserBilling(r, user)

		// If the user has a billing profile
		if err == nil {
			originalPlan := plan
			switch plan {
			case "personal":
				plan = "Personal"
			case "consultant":
				plan = "Consultant"
			case "business":
				plan = "Business"
			case "growing":
				plan = "Growing Business"
			}

			missingCard := true
			if len(userBilling.Data.CardsOnFile) > 0 {
				missingCard = false
			}

			price := billing.PlanAndDurationToPrice(plan, duration)
			cost, _ := billing.SwitchUserPlanPreview(&userBilling, duration, originalPlan)

			data := map[string]interface{}{
				"missingCard": missingCard,
				"price":       price,
				"plan":        plan,
				"duration":    duration,
				"userEmail":   user.Data.Email,
				"difference":  cost,
			}

			t := template.New("switch-confirmation.html")
			t, _ = t.ParseFiles("billing/switch-confirmation.html")
			t.Execute(w, data)
		} else {
			// If the user does not have billing profile that means that they
			// have not started their trial yet.
			http.Redirect(w, r, "/api/billing/plans/trial", 302)
			return
		}
	}
}

func ChoosePlanHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		plan := r.FormValue("plan")
		duration := r.FormValue("duration")

		// To check if there is a user logged in
		user, err := apiControllers.GetCurrentUser(r)

		if r.URL.Query().Get("next") != "" {
			session, _ := store.Get(r, "sess")
			session.Values["next"] = r.URL.Query().Get("next")
			session.Save(r, w)

			// If there is a next and the user has not been logged in
			if err != nil {
				log.Printf("%v", err)
				http.Redirect(w, r, r.URL.Query().Get("next"), 302)
				return
			}
		}

		// If there is no next and the user is not logged in
		if err != nil {
			log.Printf("%v", err)
			http.Redirect(w, r, "https://tabulae.newsai.co/", 302)
			return
		}

		userBilling, err := apiControllers.GetUserBilling(r, user)

		// If the user has a billing profile
		if err == nil {
			switch plan {
			case "personal":
				plan = "Personal"
			case "consultant":
				plan = "Consultant"
			case "business":
				plan = "Business"
			case "growing":
				plan = "Growing Business"
			}

			missingCard := true
			if len(userBilling.Data.CardsOnFile) > 0 {
				missingCard = false
			}

			price := billing.PlanAndDurationToPrice(plan, duration)

			data := map[string]interface{}{
				"missingCard": missingCard,
				"price":       price,
				"plan":        plan,
				"duration":    duration,
				"userEmail":   user.Data.Email,
			}

			t := template.New("confirmation.html")
			t, _ = t.ParseFiles("billing/confirmation.html")
			t.Execute(w, data)
		} else {
			// If the user does not have billing profile that means that they
			// have not started their trial yet.
			http.Redirect(w, r, "/api/billing/plans/trial", 302)
			return
		}
	}
}

func CheckCouponValid() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		coupon := r.FormValue("coupon")
		duration := r.FormValue("duration")

		if coupon == "" {
			nError.ReturnError(w, http.StatusInternalServerError, "Coupon error", "Please enter a coupon")
			return
		}

		coupon = strings.ToUpper(coupon)

		if coupon == "FAVORITES" && duration == "annually" {
			nError.ReturnError(w, http.StatusInternalServerError, "Coupon error", "Sorry - you can't use this coupon code on a yearly plan. Please switch the monthly one to use this!")
			return
		}

		if coupon == "PRCOUTURE" && duration == "annually" {
			nError.ReturnError(w, http.StatusInternalServerError, "Coupon error", "Sorry - you can't use this coupon code on a yearly plan. Please switch the monthly one to use this!")
			return
		}

		if coupon == "CURIOUS" && duration == "annually" {
			nError.ReturnError(w, http.StatusInternalServerError, "Coupon error", "Sorry - you can't use this coupon code on a yearly plan. Please switch the monthly one to use this!")
			return
		}

		if coupon == "PRCONSULTANTS" && duration == "annually" {
			nError.ReturnError(w, http.StatusInternalServerError, "Coupon error", "Sorry - you can't use this coupon code on a yearly plan. Please switch the monthly one to use this!")
			return
		}

		percentageOff, err := billing.GetCoupon(coupon)

		if err == nil {
			val := struct {
				PercentageOff uint64
			}{
				PercentageOff: percentageOff,
			}
			err = ffjson.NewEncoder(w).Encode(val)
		}

		if err != nil {
			log.Printf("%v", err)
			nError.ReturnError(w, http.StatusInternalServerError, "Coupon error", err.Error())
		}

		return
	}
}

func ConfirmPlanHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		plan := r.FormValue("plan")
		duration := r.FormValue("duration")
		coupon := r.FormValue("coupon")

		// To check if there is a user logged in
		user, err := apiControllers.GetCurrentUser(r)

		if r.URL.Query().Get("next") != "" {
			session, _ := store.Get(r, "sess")
			session.Values["next"] = r.URL.Query().Get("next")
			session.Save(r, w)

			// If there is a next and the user has not been logged in
			if err != nil {
				log.Printf("%v", err)
				http.Redirect(w, r, r.URL.Query().Get("next"), 302)
				return
			}
		}

		// If there is no next and the user is not logged in
		if err != nil {
			log.Printf("%v", err)
			http.Redirect(w, r, "https://tabulae.newsai.co/", 302)
			return
		}

		userBilling, err := apiControllers.GetUserBilling(r, user)

		// If the user has a billing profile
		if err == nil {
			originalPlan := plan
			switch plan {
			case "Personal":
				plan = "personal"
			case "Consultant":
				plan = "consultant"
			case "Business":
				plan = "business"
			case "Growing Business":
				plan = "growing"
			}

			err = billing.AddPlanToUser(r, user, &userBilling, plan, duration, coupon, originalPlan)
			hasError := false
			errorMessage := ""
			if err != nil {
				hasError = true
				// Return error to the "confirmation" page
				errorMessage = err.Error()
				log.Printf("%v", err)
			}

			data := map[string]interface{}{
				"plan":         originalPlan,
				"duration":     duration,
				"hasError":     hasError,
				"errorMessage": errorMessage,
				"userEmail":    user.Data.Email,
			}

			t := template.New("receipt.html")
			t, _ = t.ParseFiles("billing/receipt.html")
			t.Execute(w, data)
		} else {
			// If the user does not have billing profile that means that they
			// have not started their trial yet.
			http.Redirect(w, r, "/api/billing/plans/trial", 302)
			return
		}
	}
}

func BillingPageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := apiControllers.GetCurrentUser(r)

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

		userBilling, err := apiControllers.GetUserBilling(r, user)

		// If the user has a billing profile
		if err == nil {
			switch userBilling.Data.StripePlanId {
			case "bronze", "personal":
				userBilling.Data.StripePlanId = "Personal"
			case "aluminum", "consultant":
				userBilling.Data.StripePlanId = "Consultant"
			case "silver-1", "silver", "business":
				userBilling.Data.StripePlanId = "Business"
			case "gold-1", "gold", "growing":
				userBilling.Data.StripePlanId = "Growing Business"
			}

			customerBalance, _ := billing.GetCustomerBalance(&userBilling)
			userPlanExpires := userBilling.Data.Expires.AddDate(0, 0, -1).Format("2006-01-02")

			userbillingHistory, _ := billing.GetCustomerBillingHistory(&userBilling)
			log.Printf("%v", userbillingHistory)

			data := map[string]interface{}{
				"userBillingPlanExpires": userPlanExpires,
				"userBilling":            userBilling,
				"userEmail":              user.Data.Email,
				"userActive":             user.Data.IsActive,
				"userBalance":            customerBalance,
				"userbillingHistory":     userbillingHistory,
				csrf.TemplateTag:         csrf.TemplateField(r),
			}

			t := template.New("billing.html")
			t, _ = t.ParseFiles("billing/billing.html")
			t.Execute(w, data)
		} else {
			// If the user does not have billing profile that means that they
			// have not started their trial yet.
			http.Redirect(w, r, "/api/billing/plans/trial", 302)
			return
		}
	}
}

func PaymentMethodsPageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := apiControllers.GetCurrentUser(r)

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

		userBilling, err := apiControllers.GetUserBilling(r, user)

		// If the user has a billing profile
		if err == nil {
			cards, err := billing.GetUserCards(&userBilling)
			if err != nil {
				cards = []billing.Card{}
			}

			userFullName := strings.Join([]string{user.Data.FirstName, user.Data.LastName}, " ")

			data := map[string]interface{}{
				"userEmail":      user.Data.Email,
				"userCards":      cards,
				"userFullName":   userFullName,
				"cardsOnFile":    len(userBilling.Data.CardsOnFile),
				csrf.TemplateTag: csrf.TemplateField(r),
			}

			t := template.New("payments.html")
			t, _ = t.ParseFiles("billing/payments.html")
			t.Execute(w, data)

		} else {
			// If the user does not have billing profile that means that they
			// have not started their trial yet.
			http.Redirect(w, r, "/api/billing/plans/trial", 302)
			return
		}
	}
}

func PaymentMethodsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := apiControllers.GetCurrentUser(r)

		stripeToken := r.FormValue("stripeToken")

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

		userBilling, err := apiControllers.GetUserBilling(r, user)
		if err != nil {
			log.Printf("%v", err)
			http.Redirect(w, r, "/api/billing/plans/trial", 302)
			return
		}

		err = billing.AddPaymentsToCustomer(&userBilling, stripeToken)

		// Throw error message to user
		if err != nil {
			log.Printf("%v", err)
			http.Redirect(w, r, "/api/billing/payment-methods?error="+err.Error(), 302)
			return
		}

		http.Redirect(w, r, "/api/billing/payment-methods", 302)
		return
	}
}
