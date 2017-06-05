package controllers

import (
	"errors"
	"net/http"

	"golang.org/x/net/context"

	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"

	"github.com/qedus/nds"

	"github.com/news-ai/api/models"
)

/*
* Public methods
 */

/*
* Get methods
 */

func GetToken(c context.Context, r *http.Request, token string) (models.UserToken, error) {
	query := datastore.NewQuery("UserToken").Filter("Token =", token)
	ks, err := query.KeysOnly().GetAll(c, nil)
	if err != nil {
		log.Errorf(c, "%v", err)
		return models.UserToken{}, err
	}

	var tokens []models.UserToken
	tokens = make([]models.UserToken, len(ks))
	err = nds.GetMulti(c, ks, tokens)
	if err != nil {
		log.Infof(c, "%v", err)
		return models.UserToken{}, err
	}

	if len(tokens) > 0 {
		tokens[0].Format(ks[0], "usertokens")
		return tokens[0], nil
	}

	return models.UserToken{}, errors.New("No usertoken by the field token")
}

func GetTokensForUser(c context.Context, r *http.Request, userId int64, isUsed bool) ([]models.UserToken, error) {
	query := datastore.NewQuery("UserToken").Filter("CreatedBy =", userId).Filter("IsUsed =", isUsed)
	ks, err := query.KeysOnly().GetAll(c, nil)
	if err != nil {
		log.Errorf(c, "%v", err)
		return []models.UserToken{}, err
	}

	var tokenIds []models.UserToken
	tokenIds = make([]models.UserToken, len(ks))
	err = nds.GetMulti(c, ks, tokenIds)
	if err != nil {
		log.Infof(c, "%v", err)
		return []models.UserToken{}, err
	}

	for i := 0; i < len(tokenIds); i++ {
		tokenIds[i].Format(ks[i], "usertokens")
	}

	return tokenIds, nil
}
