package rql

import (
	"errors"
	"reflect"
	"strings"

	"github.com/davecgh/go-spew/spew"
)

var (
	errorValueNotFoundForVariable = errors.New("cannot find value for variable in filter expression ")
)

func FlattenAllFields(iface interface{}) []reflect.StructField {
	fields := make([]reflect.StructField, 0)
	ifv := reflect.ValueOf(iface)
	ift := reflect.TypeOf(iface)

	for i := 0; i < ift.NumField(); i++ {

		v := ifv.Field(i)
		t := ift.Field(i)
		switch v.Kind() {
		case reflect.Struct:
			if v.Type().Name() == "Time" {
				fields = append(fields, t)
			} else {
				fields = append(fields, FlattenAllFields(v.Interface())...)
			}

		default:
			fields = append(fields, t)
		}
	}

	return fields
}

// GetSchemaFromTaggedEntity : fetches a schema object from a model (panics on error!) , your models must all have json tags for this function
func GetSchemaFromTaggedEntity(model interface{}, filterColTag string) *Schema {
	var schemaOut Schema
	schemaOut.supportedColumns = make(map[string]*FilterableEntity)
	fields := FlattenAllFields(model)
	for _, t := range fields {
		internalVal, existsInternal := t.Tag.Lookup(filterColTag)
		if existsInternal {
			_, isNotFilterable := t.Tag.Lookup("not_filterable")
			_, isNotFuzzySearch := t.Tag.Lookup("not_searchable")
			schemaOut.supportedColumns[internalVal] = &FilterableEntity{
				IsFilterable:       !isNotFilterable,
				IsFuzzySearchable:  !isNotFuzzySearch,
				Type:               resolveJavaScriptType(t),
				ColumnNameInternal: internalVal,
			}
		}
	}
	spew.Dump(schemaOut)
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
	case "time.Time":
		return "Date"
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
