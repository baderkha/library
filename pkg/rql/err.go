package rql

import "strings"

// IsRQLError : checks if error is an rql based error (ie will allow you to distinguis a user input error)
func IsRQLError(err error) bool {
	return strings.Contains(err.Error(), "RQL :")
}
