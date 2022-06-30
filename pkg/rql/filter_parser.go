package rql

type ISQLFilterParser = IFilterParser[SQLOutput]

// IFilterParser : abstract filter expression parser
type IFilterParser [t any]interface {
	// Parse :  Filter Expression and output it to a result object
	Parse(expression *FilterExpression, schema *Schema) (out *t, err error)
}

type IFilterValidator interface{
	//	Validate : validate filter expression
	Validate(expression *FilterExpression, schema *Schema) error
}
