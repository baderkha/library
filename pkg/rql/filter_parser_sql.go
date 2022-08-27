package rql

import (
	"errors"
	"strings"

	"github.com/baderkha/library/pkg/err"
)

var (

	// composed errors

	SQLErrBoolOp                        = err.Compose("RQL : SQL : FilterParser : unsupported boolean operation `%s` expected either `%s`,`%s`")
	SQLErrorColumnNotFound              = err.Compose("RQL : SQL : FilterParser : Column `%s` does not exist")
	SQLErrOperatorForColumnNotSupported = err.Compose("RQL : SQL : FilterParser : this operator %s is not supported")

	// static errors
	SQLErrVariables = errors.New("RQL : SQL : FilterParser : you cannot have variables and values set or null . it's either one or the other being set or null")
)

type SQLOutput struct {
	Query string
	Args  []interface{}
}

type sQLOperator struct {
	Name       string
	SQL        string
	MultiValue bool
}

type sQLOperators []sQLOperator

func (s *sQLOperators) getOperator(op string) (string, bool, error) {
	for _, o := range *s {
		if op == o.Name {
			return o.SQL, false, nil
		}
	}
	return "", false, SQLErrOperatorForColumnNotSupported(op)
}

// filter operators for v2
var (
	filterOps2 = sQLOperators{
		sQLOperator{Name: filterFuzzy, SQL: "LIKE ?", MultiValue: false},
		sQLOperator{Name: filterLike, SQL: "like ? ", MultiValue: false},
		sQLOperator{Name: filterGt, SQL: "> ? ", MultiValue: false},
		sQLOperator{Name: filterGe, SQL: ">= ? ", MultiValue: false},
		sQLOperator{Name: filterGe, SQL: "< ? ", MultiValue: false},
		sQLOperator{Name: filterLe, SQL: "<= ? ", MultiValue: false},
		sQLOperator{Name: filterEq, SQL: "= ? ", MultiValue: false},
		sQLOperator{Name: filterNe, SQL: "<> ? ", MultiValue: false},
		sQLOperator{Name: filterIn, SQL: "IN (?) ", MultiValue: true},
		sQLOperator{Name: filterNin, SQL: "NOT IN (?) ", MultiValue: true},
	}
	errorColumnNotFound = errors.New("column not found")
)

const (
	ANDOperator = "AND"
	OROperator  = "OR"
)

func (s *SQLBaseFilterParser) resolveBoolOp(boolop string) (string, error) {
	switch boolop {
	case ANDOperator:
		return ANDOperator, nil
	case OROperator:
		return OROperator, nil
	default:
		return "", SQLErrBoolOp(boolop, ANDOperator, OROperator)
	}
}

var _ IFilterParser[SQLOutput] = &SQLBaseFilterParser{}
var _ IFilterValidator = &SQLBaseFilterParser{}

func NewSQLFilterValidator() IFilterValidator {
	return &SQLBaseFilterParser{}
}

type SQLBaseFilterParser struct {
}

func (s *SQLBaseFilterParser) Validate(expression *FilterExpression, schema *Schema) error {
	properties := expression.Properties
	_, err := s.resolveBoolOp(expression.BinaryOperation)
	if err != nil {
		return err
	}
	// base case
	if len(properties) == 0 {
		return nil
	}

	for i := 0; i < len(properties); i++ {
		filter := properties[i]
		if filter.Column != "" && filter.Op != "" {
			hasCol := schema.DoesColExist(filter.Column)
			if !hasCol {
				return SQLErrorColumnNotFound(filter.Column)
			}
			_, _, err := filterOps2.getOperator(filter.Op)
			if err != nil {
				return err
			}

			if (filter.Value != nil && filter.Variable != nil) ||
				(filter.Value == nil && filter.Variable == nil) {
				return SQLErrVariables
			}

		} else if filter.Properties != nil && len(filter.Properties) > 0 {
			return s.Validate(filter, schema)
		}
	}
	return nil
}

func (s SQLBaseFilterParser) Parse(expression *FilterExpression, schema *Schema) (*SQLOutput, error) {
	var parseOut SQLOutput
	sql, args, err := s.ParseRaw(expression, schema)
	parseOut.Query = sql
	parseOut.Args = args
	return &parseOut, err
}

func (s SQLBaseFilterParser) ParseRaw(expression *FilterExpression, schema *Schema) (string, []interface{}, error) {
	if expression == nil {
		return "", nil, nil
	}
	var sqlAr []string
	var args []interface{}
	properties := expression.Properties

	boolOp, err := s.resolveBoolOp(expression.BinaryOperation)
	if err != nil {
		return "", nil, err
	}
	// base case
	if len(properties) == 0 {
		return "", args, nil
	}

	for i := 0; i < len(properties); i++ {
		filter := properties[i]
		if filter.Column != "" && filter.Op != "" && filter.Value != "" {
			hasCol := schema.DoesColExist(filter.Column)
			if !hasCol {
				return "", nil, SQLErrorColumnNotFound(filter.Column)
			}

			op, _, err := filterOps2.getOperator(filter.Op)
			if err != nil {
				return "", nil, err
			}

			sqlAr = append(sqlAr, " "+schema.GetColumnInternalName(filter.Column)+" "+op)
			args = append(args, filter.Value)
		} else if filter.Properties != nil && len(filter.Properties) > 0 {
			childSQL, childArgs, err := s.ParseRaw(filter, schema)
			if err != nil {
				return "", args, err
			}
			sqlAr = append(sqlAr, childSQL)
			args = append(args, childArgs...)
		}
	}
	return " ( " + strings.Join(sqlAr, " "+boolOp+" ") + " ) ", args, nil
}
