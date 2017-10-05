package controllers

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/mail"

	"github.com/pquerna/ffjson/ffjson"

	"github.com/news-ai/api-v1/db"
	"github.com/news-ai/api-v1/models"

	"github.com/news-ai/tabulae-v1/emails"

	"github.com/news-ai/web/utilities"
)

/*
* Private
 */

/*
* Private methods
 */

/*
* Get methods
 */

func generateTokenAndEmail(r *http.Request, invite models.Invite) (models.UserInviteCode, error) {
	currentUser, err := GetCurrentUser(r)
	if err != nil {
		log.Printf("%v", err)
		return models.UserInviteCode{}, err
	}

	validEmail, err := mail.ParseAddress(invite.Email)
	if err != nil {
		invalidEmailError := errors.New("Email user has entered is incorrect")
		log.Printf("%v", invalidEmailError)
		return models.UserInviteCode{}, invalidEmailError
	}

	userInviteCodes := []models.UserInviteCode{}
	err = db.DB.Model(&userInviteCodes).Where("email = ?", validEmail.Address).Select()
	if err != nil {
		log.Printf("%v", err)
		return models.UserInviteCode{}, err
	}

	if len(userInviteCodes) > 0 {
		hasBeenUsed := false

		for i := 0; i < len(userInviteCodes); i++ {
			userInviteCodes[i].Type = "invites"

			if userInviteCodes[i].IsUsed {
				hasBeenUsed = true
			}
		}

		if hasBeenUsed {
			invitedAlreadyError := errors.New("User has already been invited to the NewsAI platform")
			log.Printf("%v", invitedAlreadyError)
			return models.UserInviteCode{}, invitedAlreadyError
		}
	}

	// Check if the user is already a part of the platform
	_, err = GetUserByEmail(validEmail.Address)
	if err == nil {
		userExistsError := errors.New("User already exists on the NewsAI platform")
		log.Printf("%v", userExistsError)
		return models.UserInviteCode{}, userExistsError
	}

	referralCode := models.UserInviteCode{}
	referralCode.Email = validEmail.Address
	referralCode.InviteCode = utilities.RandToken()
	_, err = referralCode.Create(r, currentUser)
	if err != nil {
		log.Printf("%v", err)
		return models.UserInviteCode{}, err
	}

	// Email this person with the referral code
	inviteUserEmailErr := emails.InviteUser(currentUser.Data, validEmail.Address, referralCode.InviteCode, invite.PersonalNote)
	if inviteUserEmailErr != nil {
		// Redirect user back to login page
		log.Printf("%v", "Invite email was not sent for "+validEmail.Address)
		log.Printf("%v", err)
		inviteEmailError := errors.New("Could not send invite email. We'll fix this soon!")
		return models.UserInviteCode{}, inviteEmailError
	}

	return referralCode, nil
}

/*
* Public methods
 */

/*
* Get methods
 */

func GetInvites(r *http.Request) ([]models.UserInviteCode, interface{}, int, int, error) {
	currentUser, err := GetCurrentUser(r)
	if err != nil {
		log.Printf("%v", err)
		return []models.UserInviteCode{}, nil, 0, 0, err
	}

	userInviteCodes := []models.UserInviteCode{}
	err = db.DB.Model(&userInviteCodes).Where("created_by = ?", currentUser.Id).Where("is_used = ?", true).Select()
	if err != nil {
		log.Printf("%v", err)
		return []models.UserInviteCode{}, nil, 0, 0, err
	}

	for i := 0; i < len(userInviteCodes); i++ {
		userInviteCodes[i].Type = "invites"
	}

	return userInviteCodes, nil, len(userInviteCodes), 0, nil
}

func GetInviteFromInvitationCode(r *http.Request, invitationCode string) (models.UserInviteCode, error) {
	userInviteCode := models.UserInviteCode{}
	err := db.DB.Model(&userInviteCode).Where("invite_code = ?", invitationCode).Select()
	if err != nil {
		log.Printf("%v", err)
		return models.UserInviteCode{}, err
	}

	if userInviteCode.InviteCode != "" {
		userInviteCode.Type = "invites"
		return userInviteCode, nil
	}

	return models.UserInviteCode{}, errors.New("No invitation by that code")
}

/*
* Create methods
 */

func CreateInvite(r *http.Request) (models.UserInviteCode, interface{}, error) {
	buf, _ := ioutil.ReadAll(r.Body)
	decoder := ffjson.NewDecoder()
	var invite models.Invite
	err := decoder.Decode(buf, &invite)
	if err != nil {
		log.Printf("%v", err)
		return models.UserInviteCode{}, nil, err
	}

	userInvite, err := generateTokenAndEmail(r, invite)
	if err != nil {
		return models.UserInviteCode{}, nil, err
	}

	userInvite.Type = "invites"
	return userInvite, nil, nil
}
