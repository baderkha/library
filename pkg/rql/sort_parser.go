package rql

type ISQLSortParser = ISortParser[SQLSortOutput]

// ISortParser : parses sort expression
type ISortParser[t any] interface {
	// Parse : Sort expression parser
	Parse(expression *SortExpression, schema *Schema) (out *t, err error)
}
