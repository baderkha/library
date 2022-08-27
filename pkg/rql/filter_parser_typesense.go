package rql

import (
	"errors"
	"fmt"
	"strings"

	"github.com/baderkha/library/pkg/conditional"
	"github.com/baderkha/library/pkg/err"
	"github.com/baderkha/typesense"
)

const (
	isTypesenseFuzzySearch = "fuzzy_search"
	unsupportedBaseFilter  = "unsupported"
)

var (

	// composed errors
	TsErrUnknownOperation                         = err.Compose("RQL : TypeSense : FilterParser : Operation '%s' is unknown ")
	TsErrUnsupportedBaseOperation                 = err.Compose("RQL : TypeSense : FilterParser : Although Operation '%s' is a base filter , typesense does not support it as of this time")
	TsErrorExpectedThisFilterToHaveADifferentType = err.Compose("RQL : TypeSense : FilterParser : Expected this expression `%s` to have value of type `%s`")
	TsErrorThisColumnDoesNotSupportSearching      = err.Compose("RQL : TypeSense : FilterParser : Column `%s` does not support searching")
	TsErrorThisColumnDoesNotSupportFiltering      = err.Compose("RQL : TypeSense : FilterParser : Column `%s` does not support filtering")
	TsErrorColumnNotFond                          = err.Compose("RQL : TypeSense : FilterParser : Column `%s` does not exist")
	TsErrorColumnNotFuzzySearchable               = err.Compose("RQL : TypeSense : FilterParser : Column `%s` is not fuzzy searchable , you must have an index on it")

	// static errors
	TsErrTypesenseCannotDoOrs                               = errors.New("RQL : TypeSense : FilterParser : typesense cannot do or logic")
	TsErrorYourLikeOperationsShouldAllHaveTheSameSearchTerm = errors.New("RQL : TypeSense : FilterParser : Your Like Operations must all have the same values")
	TsErrTypesenseCannotHaveMoreThan1Level                  = errors.New("RQL : TypeSense : FilterParser : typesense cannot have more than 1 level of filter nesting")

	typesenseFilteOps = map[string]string{
		filterLike:  unsupportedBaseFilter,
		filterFuzzy: isTypesenseFuzzySearch,
		filterGt:    ":>",
		filterGe:    ":>=",
		filterLt:    ":<",
		filterLe:    ":<=",
		filterEq:    ":=",
		filterNe:    ":!=",
		filterIn:    ":",
		filterNin:   ":!",
	}
)

type FilterParserTypeSense struct {
}

func (f *FilterParserTypeSense) parseOperation(operation string) (operat string, isMultiValueOperator bool, err error) {
	op := typesenseFilteOps[operation]
	switch op {
	case "":
		return "", false, TsErrUnknownOperation(operation)
	case unsupportedBaseFilter:
		return "", false, TsErrUnsupportedBaseOperation(operation)
	}
	return op, operation == filterIn || operation == filterNin, nil

}

func (f *FilterParserTypeSense) isPropertyNestingMoreThan1(property *FilterExpression) bool {
	return len(property.Properties) > 0
}

func (f *FilterParserTypeSense) Parse(expression *FilterExpression, schema *Schema) (out *typesense.SearchParameters, err error) {
	var (
		fuzzySearchByFields []string
		fuzzySearchByTerm   string
		filterByArgs        []string
	)
	properties := expression.Properties
	for _, prop := range properties {
		if f.isPropertyNestingMoreThan1(prop) {
			return nil, TsErrTypesenseCannotHaveMoreThan1Level
		}
		if !schema.DoesColExist(prop.Column) {
			return nil, errorColumnNotFound
		}

		op, isMulti, err := f.parseOperation(prop.Op)
		if err != nil {
			return nil, err
		}

		if op == isTypesenseFuzzySearch {
			if !schema.CheckTagExists(prop.Column, typesense.TagIndex) {
				return nil, TsErrorColumnNotFuzzySearchable(prop.Column)
			}
			fSearchTerm, stringCastable := prop.Value.(string)
			// not string
			if !stringCastable {
				return nil, TsErrorExpectedThisFilterToHaveADifferentType(
					fmt.Sprintf("%s:%s:%s", prop.Column, prop.Op, prop.Value),
					"string",
				)
			}

			if fuzzySearchByTerm == "" {
				fuzzySearchByTerm = fSearchTerm
			} else if fSearchTerm != fuzzySearchByTerm {
				return nil, TsErrorYourLikeOperationsShouldAllHaveTheSameSearchTerm
			}
			fuzzySearchByFields = append(fuzzySearchByFields, prop.Column)

		} else {
			filterByArgs = conditional.Ternary(
				isMulti,
				append(filterByArgs, fmt.Sprintf("%s%s[%s]", prop.Column, op, prop.Value)),
				append(filterByArgs, fmt.Sprintf("%s%s%s", prop.Column, op, prop.Value)),
			)
		}

	}

	search := typesense.
		NewSearchParams().
		AddQueryBy(strings.Join(fuzzySearchByFields, ",")).
		AddSearchTerm(fuzzySearchByTerm).
		AddFilterBy(strings.Join(filterByArgs, "&&"))

	return search, nil
}

func (f *FilterParserTypeSense) Validate(expression *FilterExpression, schema *Schema) (err error) {
	_, err = f.Parse(expression, schema)
	return err
}
