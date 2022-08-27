package rql

import (
	"fmt"
	"strings"

	"github.com/baderkha/library/pkg/ptr"
)

var _ ITypeSenseSortParser = &SortParserTypesense{}

type SortParserTypesense struct {
}

// Parse : parse an expression and turn it into sql expression
func (s SortParserTypesense) Parse(expression *SortExpression, schema *Schema) (out *string, err error) {
	var args []string
	for k, v := range expression.sortMap {
		if !schema.DoesColExist(k) {
			return nil, ErrSortColumnDoesntExist(k)
		}
		if v != DESC && v != ASC {
			return nil, ErrBadSortExpressionValue
		}
		args = append(args, fmt.Sprintf("%s:%s", k, strings.ToLower(v)))
	}
	return ptr.Get(strings.Join(args, ",")), nil
}
