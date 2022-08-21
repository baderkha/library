package sso

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/baderkha/library/pkg/store/entity"
	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
)

var _ Handler = &Google{}

const (
	GoogleHandlerErrWrap = "GOOGLE SSO HANDLER ERROR :"
)

var (
	ErrorInvalidGoogleJWTClaim = fmt.Errorf("%s %s", GoogleHandlerErrWrap, "Could not Verify JWT Claim")
	ErrorExpiredGoogleJWTClaim = fmt.Errorf("%s %s", GoogleHandlerErrWrap, "Auth Token is Expired")
	ErrorMissingGoogleRSAKey   = fmt.Errorf("%s %s", GoogleHandlerErrWrap, "RSA Key could not be found")
)

// GoogleClaims : google jwt claims we get back from the token
type GoogleClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	FirstName     string `json:"given_name"`
	LastName      string `json:"family_name"`
	jwt.StandardClaims
}

// Google :
// A 0 depedency Google SSO handler that adapts perfectly to any Framework as long as you can provide
// the http.Request Object
type Google struct {
	ClientID string
}

func (g *Google) VerifyUser(req http.Header) (res *entity.Account, err error) {
	authToken := req.Get("Authorization")
	claim, err := g.ValidateGoogleJWT(authToken)
	if err != nil {
		fmt.Println(err.Error())
		return nil, ErrorUnauthorized
	}
	return &entity.Account{
		AccountPublic: entity.AccountPublic{
			Base: entity.Base{
				ID: claim.Email,
			},
			Email:      claim.Email,
			IsSSO:      true,
			SSOType:    HandlerTypeGoogle,
			IsVerified: claim.EmailVerified,
		},
		Password: "",
	}, nil
}

func (g *Google) ValidateGoogleJWT(tokenString string) (GoogleClaims, error) {
	claimsStruct := GoogleClaims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		&claimsStruct,
		func(token *jwt.Token) (interface{}, error) {
			pem, err := g.getGooglePublicKey(fmt.Sprintf("%s", token.Header["kid"]))
			if err != nil {
				return nil, err
			}
			key, err := jwt.ParseRSAPublicKeyFromPEM([]byte(pem))
			if err != nil {
				return nil, err
			}
			return key, nil
		},
	)
	if err != nil {
		return GoogleClaims{}, err
	}

	claims, ok := token.Claims.(*GoogleClaims)
	if !ok {
		return GoogleClaims{}, errors.New("Invalid Google JWT")
	}

	if claims.Issuer != "accounts.google.com" && claims.Issuer != "https://accounts.google.com" {
		return GoogleClaims{}, ErrorInvalidGoogleJWTClaim
	}

	if claims.Audience != g.ClientID {
		return GoogleClaims{}, ErrorInvalidGoogleJWTClaim
	}

	if claims.ExpiresAt < time.Now().UTC().Unix() {
		return GoogleClaims{}, ErrorExpiredGoogleJWTClaim
	}

	return *claims, nil
}

func (g *Google) getGooglePublicKey(keyID string) (string, error) {
	resp, err := http.Get("https://www.googleapis.com/oauth2/v1/certs")
	if err != nil {
		return "", err
	}
	dat, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	myResp := map[string]string{}
	err = json.Unmarshal(dat, &myResp)
	if err != nil {
		return "", err
	}
	key, ok := myResp[keyID]
	if !ok {
		return "", ErrorMissingGoogleRSAKey
	}
	return key, nil
}
