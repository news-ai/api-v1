package models

import (
	"log"
	"strings"
	"time"

	"github.com/news-ai/api-v1/db"
)

type UserFeedback struct {
	ReasonNotPurchase  string `json:"reason"`
	FeedbackAfterTrial string `json:"feedback"`
}

type UserPlan struct {
	PlanName string `json:"planname"`

	EmailAccounts      int `json:"emailaccounts"`
	DailyEmailsAllowed int `json:"dailyemailsallowed"`

	EmailsSentToday int `json:"emailssenttoday"`

	OnTrial bool `json:"ontrial"`
}

type UserNewPlan struct {
	Plan     string `json:"plan"`
	Duration string `json:"duration"`
	Coupon   string `json:"coupon"`
}

type UserLiveToken struct {
	Token   string    `json:"token"`
	Expires time.Time `json:"expires"`
}

type User struct {
	Base

	GoogleId string `json:"googleid"`

	Email     string `json:"email"`
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`

	Emails []string `json:"sendgridemails"`

	EmailProvider string `json:"emailprovider" datastore:"-"`

	EmailAlias string `json:"-"`

	Password []byte `json:"-"`
	ApiKey   string `json:"-"`

	Employers []int64 `json:"employers" apiModel:"Agency"`

	ResetPasswordCode      string `json:"-"`
	ConfirmationCode       string `json:"-"`
	ConfirmationCodeBackup string `json:"-"`

	LastLoggedIn time.Time `json:"-"`

	// Social network settings
	LinkedinId      string `json:"-"`
	LinkedinAuthKey string `json:"-"`

	InstagramId      string `json:"-"`
	InstagramAuthKey string `json:"-"`

	InvitedBy int64 `json:"-"`

	AgreeTermsAndConditions bool `json:"-"`
	EmailConfirmed          bool `json:"emailconfirmed"`

	GetDailyEmails bool `json:"getdailyemails"`

	BillingId int64 `json:"-"`
	TeamId    int64 `json:"teamid"`

	// Email settings
	EmailSignature  string   `json:"emailsignature" datastore:",noindex"`
	EmailSignatures []string `json:"emailsignatures" datastore:",noindex"`

	Gmail           bool      `json:"gmail"`
	AccessToken     string    `json:"-"`
	GoogleCode      string    `json:"-"`
	GoogleExpiresIn time.Time `json:"-"`
	RefreshToken    string    `json:"-"`
	TokenType       string    `json:"-"`

	Outlook             bool      `json:"outlook"`
	OutlookEmail        string    `json:"outlookusername"`
	OutlookAccessToken  string    `json:"-"`
	OutlookExpiresIn    time.Time `json:"-"`
	OutlookRefreshToken string    `json:"-"`
	OutlookTokenType    string    `json:"-"`

	// Access Token for Live Notifications
	LiveAccessToken       string    `json:"-"`
	LiveAccessTokenExpire time.Time `json:"-"`

	ExternalEmail bool  `json:"externalemail"`
	EmailSetting  int64 `json:"emailsetting"`

	SMTPValid    bool   `json:"smtpvalid"`
	SMTPUsername string `json:"smtpusername"`
	SMTPPassword []byte `json:"-"`

	UseSparkPost bool `json:"-"`

	PromoCode string `json:"-"`

	IsAdmin bool `json:"isadmin"`

	IsActive            bool `json:"isactive"`
	IsBanned            bool `json:"isbanned"`
	MediaDatabaseAccess bool `json:"mediadatabaseaccess"`

	TrialFeedback bool `json:"trialfeedback"`

	UserType string `json:"-"` // Journalist or PR
	Profile  int64  `json:"-"`

	EnhanceCredits int `json:"-"`
}

type UserPostgres struct {
	Id int64

	Data User
}

/*
* Public methods
 */

/*
* Create methods
 */

func (u *UserPostgres) Normalize() (*UserPostgres, error) {
	u.Data.Email = strings.ToLower(u.Data.Email)
	u.Data.FirstName = strings.Title(u.Data.FirstName)
	u.Data.LastName = strings.Title(u.Data.LastName)
	return u, nil
}

func (u *UserPostgres) Create() (*UserPostgres, error) {
	// Create user
	u.Data.IsAdmin = false
	u.Data.GetDailyEmails = true
	u.Data.Created = time.Now()

	u.Normalize()

	_, err := db.DB.Model(u).Returning("*").Insert()
	return u, err
}

/*
* Update methods
 */

// Function to save a new user into App Engine
func (u *UserPostgres) Save() (*UserPostgres, error) {
	u.Data.Updated = time.Now()
	_, err := db.DB.Model(u).Set("data = ?data").Where("id = ?id").Returning("*").Update()
	return u, err
}

func (u *UserPostgres) ConfirmEmail() (*UserPostgres, error) {
	u.Data.EmailConfirmed = true
	u.Data.ConfirmationCode = ""
	_, err := u.Save()
	if err != nil {
		log.Printf("%v", err)
		return u, err
	}
	return u, nil
}

func (u *UserPostgres) ConfirmLoggedIn() (*UserPostgres, error) {
	u.Data.LastLoggedIn = time.Now()
	_, err := u.Save()
	if err != nil {
		log.Printf("%v", err)
		return u, err
	}
	return u, nil
}
