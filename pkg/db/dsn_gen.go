package db

import (
	"fmt"
	"strings"
)

const (
	// DialectMYSQL : use this for mysql dsn generation
	DialectMYSQL = "MYSQL"
)

// GetDSN : generate dsn string for the correct sql dialect
func GetDSN(dialect string, host string, username string, password string, port string, database string, queryParam ...string) string {
	switch dialect {
	case DialectMYSQL:
		return fmt.Sprintf(`%s:%s@%s:%s/%s?%s`, username, password, host, port, database, strings.Join(queryParam, "&"))
	default:
		return ""
	}
}
