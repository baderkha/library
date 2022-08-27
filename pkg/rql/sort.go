package rql

import (
	"errors"
	"strings"

	"github.com/baderkha/library/pkg/err"
)

var (
	ErrBadSortExpression               = errors.New("RQL : SortExpression Malformed must be `col::<ASC|DESC>` ")
	ErrBadSortExpressionValue          = errors.New("RQL : SortExpression Malformed must be either DESC|ASC for the value")
	ErrBadSortExpressionNotSortableCol = errors.New("RQL : SortExpression Malformed col not found ")
	ErrSortColumnDoesntExist           = err.Compose("SQL : SortExpression Column `%s` Does Not Exist")

	DESC = "DESC"
	ASC  = "ASC"
)

// SortExpression : sort expression value
type SortExpression struct {
	sortMap map[string]string
}

// SortExpressionFromUserInput : sort expression from user input
func SortExpressionFromUserInput(sortStr string) (*SortExpression, error) {

	if sortStr == "" {
		return &SortExpression{
			sortMap: make(map[string]string, 0),
		}, nil
	}
	exprAr := strings.Split(sortStr, ",")
	sortMap := make(map[string]string, len(exprAr))

	for _, item := range exprAr {
		kv := strings.Split(item, "::")
		if len(kv) != 2 {
			return nil, ErrBadSortExpression
		}
		if kv[1] != DESC && kv[1] != ASC {
			return nil, ErrBadSortExpressionValue
		}
		sortMap[kv[0]] = kv[1]
	}
	return &SortExpression{
		sortMap: sortMap,
	}, nil

}
