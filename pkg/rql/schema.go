package rql

import (
	"errors"
	"reflect"
	"strings"
)

var (
	errorValueNotFoundForVariable = errors.New("cannot find value for variable in filter expression ")
)

// GetSchemaFromTaggedEntity : fetches a schema object from a model (panics on error!) , your models must all have json tags for this function
func GetSchemaFromTaggedEntity(model interface{}, filterColTag string) *Schema {
	var schemaOut Schema
	t := reflect.TypeOf(model)
	for i := 0; i < t.NumField(); i++ {
		val, exists := t.Field(i).Tag.Lookup("json")
		internalVal, existsInternal := t.Field(i).Tag.Lookup(filterColTag)
		if exists && existsInternal {
			_, isFilterable := t.Field(i).Tag.Lookup("filterable")
			_, isFuzzySearch := t.Field(i).Tag.Lookup("searchable")
			schemaOut.supportedColumns[val] = &FilterableEntity{
				IsFilterable:       isFilterable,
				IsFuzzySearchable:  isFuzzySearch,
				Type:               resolveJavaScriptType(t.Field(i)),
				ColumnNameInternal: internalVal,
			}
		}
	}
	return &schemaOut
}

func resolveJavaScriptType(v reflect.StructField) string {
	tpe := strings.ReplaceAll(v.Type.String(), "*", "")

	switch tpe {
	case "float64":
	case "float32":
	case "int8":
	case "int16":
	case "int32":
	case "int64":
	case "int":
		return "number"
	case "string":
		return "string"
	case "bool":
		return "boolean"
	}
	return "Object"

}

type FilterableEntity struct {
	IsFilterable       bool   `json:"is_filterable"`
	IsFuzzySearchable  bool   `json:"is_fuzzy_searchable"`
	Type               string `json:"type"` // string , number , obj , ...etc
	ColumnNameInternal string `json:"-"`    // internal col name for sql
}

type Schema struct {
	supportedColumns map[string]*FilterableEntity
}

func (s *Schema) IsSortable(col string) bool {
	return s.supportedColumns[col] != nil
}

func (s *Schema) IsFilterable(col string) bool {
	return s.supportedColumns[col] != nil && s.supportedColumns[col].IsFilterable
}

func (s *Schema) IsSearchable(col string) bool {
	return s.supportedColumns[col] != nil && s.supportedColumns[col].IsFuzzySearchable
}

func (s *Schema) GetColumnInternalName(col string) string {
	return s.supportedColumns[col].ColumnNameInternal
}
