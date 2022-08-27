package rql

import (
	"fmt"
	"strings"

	"github.com/baderkha/library/pkg/conditional"
)

// SQLSortOutput : sql sort output for order by claueses
type SQLSortOutput struct {
	// FULL STRING
	RawQuery string
	// order by arrays ["'DATE DESC' , 'COL ASC'"]
	Clauses []string
}

var _ ISortParser[SQLSortOutput] = &SortParserSQL{}

// SortParserSQL : sort parser sql
type SortParserSQL struct {
}

// Parse : parse an expression and turn it into sql expression
func (s SortParserSQL) Parse(expression *SortExpression, schema *Schema) (out *SQLSortOutput, err error) {
	out = &SQLSortOutput{}
	for k, v := range expression.sortMap {
		if !schema.DoesColExist(k) {
			return nil, ErrSortColumnDoesntExist(k)
		}
		if v != DESC && v != ASC {
			return nil, ErrBadSortExpressionValue
		}
		out.Clauses = append(out.Clauses, fmt.Sprintf("%s %s", k, v))
	}
	out.RawQuery = conditional.Ternary(len(out.Clauses) > 0, fmt.Sprintf("ORDER BY %s", strings.Join(out.Clauses, ",")), "")

	return out, nil
}
