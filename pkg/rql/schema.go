package rql

import (
	"errors"
	"reflect"
	"strings"

	"github.com/tkrajina/go-reflector/reflector"
)

const (
	// RQLNoOpTag : tag label for column you do not wish to be operated on in any way
	// This can be a credentials column ...etc
	RQLNoOpTag = "rql_no_op"
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
	refl := reflector.New(model)
	fields := refl.FieldsFlattened()
	for _, t := range fields {
		noOpVal, _ := t.Tag(RQLNoOpTag)
		if noOpVal != "" {
			continue // skip
		}
		internalVal, existsInternal := t.Tag(filterColTag)
		var tags map[string]string
		tags, err := t.Tags()
		if err != nil {
			tags = make(map[string]string, 0)
		}
		if existsInternal == nil && internalVal != "" {
			schemaOut.supportedColumns[internalVal] = &FilterableEntity{
				ColumnNameInternal: internalVal,
				Tags:               tags,
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
	case "time.Time":
		return "Date"
	case "bool":
		return "boolean"
	}
	return "Object"

}

type FilterableEntity struct {
	ColumnNameInternal string `json:"-"` // internal col name for sql
	Tags               map[string]string
}

type Schema struct {
	supportedColumns map[string]*FilterableEntity
}

func (s *Schema) GetColumnInternalName(col string) string {
	return s.supportedColumns[col].ColumnNameInternal
}

func (s *Schema) DoesColExist(col string) bool {
	return s.GetColumnInternalName(col) != ""
}

func (s *Schema) CheckTagExists(col string, tag string) bool {
	return s.GetTagValue(col, tag) != ""
}

func (s *Schema) GetTagValue(col string, tag string) string {
	fe := s.supportedColumns[col]
	if fe != nil {
		return ""
	}
	return fe.Tags[tag]
}
