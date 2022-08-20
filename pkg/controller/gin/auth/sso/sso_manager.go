package sso

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/baderkha/library/pkg/store/entity"
)

var (
	// ErrorUnsupportedHandlerType : if the header sso_type returns something we don't like
	ErrorUnsupportedHandlerType = fmt.Errorf("unsupported SSO Handler , please ensure you only use the following sso types %s", supportedSSOHandlers)
	// ErrorHandlerIsNotConfiguredProperly : if one of the handlers is nil
	ErrorHandlerIsNotConfiguredProperly = fmt.Errorf("this is a backend problem , please contact admin . SSO handler is not configured properly")
)

// Manager : Acts like an adapter pattern . It adapts to the same interface
// 			 But under the hood it can call the correct sso handler implementaton ie google , facebook , apple ..etc
type Manager struct {
	handlers map[string]Handler
}

// VerifyUser : Calls the appropriate Handler to do verification
func (m *Manager) VerifyUser(req http.Header) (acc *entity.Account, err error) {

	ssoHandlerType := req.Get("sso_type")
	if !strings.Contains(supportedSSOHandlers, ssoHandlerType) {
		return nil, ErrorUnsupportedHandlerType
	}

	ssoHandlerImplementation := m.handlers[ssoHandlerType]
	if ssoHandlerImplementation == nil {
		return nil, ErrorHandlerIsNotConfiguredProperly
	}
	return ssoHandlerImplementation.VerifyUser(req)
}
