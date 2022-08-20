package sso

import (
	"errors"
	"net/http"
	"strings"

	"github.com/baderkha/library/pkg/store/entity"
)

const (
	HandlerTypeGoogle = "GOOGLE"
	//HandlerTypeFaceBook = "FaceBook"
)

var (
	supportedSSOHandlers = strings.Join(
		[]string{
			HandlerTypeGoogle,
			//HandlerTypeFaceBook,
		},
		" , ",
	)

	h Handler

	ErrorUnauthorized = errors.New("Unauthorized")
)

// Config : configuration for sso handlers
type Config struct {
	// GoogleClientID : google client id for your app
	GoogleClientID string
}

// Handler : generic sso handler
type Handler interface {
	// VerifyUser : this method shall not write to method body !!
	// use this method to authenticate a client using an sso provider
	// you're free to choose how you want your header data to look like
	VerifyUser(req http.Header) (acc *entity.Account, err error)
}

// New :
// create a new instance of an SSO Handler (this will contain all the handler types (google , apple , ...etc))
// see sso.Manager
//
//When you call the VerifyUser method , the Manager will call the appropriate handler type by checking the header
//For sso_type , if that is supported then it will execute whatever code exists for that handler
//
//IE if sso_type == "GOOGLE" then it will execute the sso.Google Handler
func New(config Config) Handler {
	if h == nil {
		h = &Manager{
			handlers: map[string]Handler{
				HandlerTypeGoogle: &Google{
					ClientID: config.GoogleClientID,
				},
			},
		}
	}
	return h
}
