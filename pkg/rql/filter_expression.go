package rql

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
)

// FilterExpression : recursive filter expression that can be used to do complex binary logic filtering
type FilterExpression struct {
	Column          string              `json:"column" mapstructure:"column"`
	Op              string              `json:"op" mapstructure:"op"`
	Value           interface{}         `json:"value" mapstructure:"value"`       // we have no idea what the value is so interface it is -- ps , i don't like that
	Variable        *string             `json:"variable" mapstructure:"variable"` // instead of a hard coded value now have it as a variable label
	Properties      []*FilterExpression `json:"properties" mapstructure:"properties"`
	BinaryOperation string              `json:"operation" mapstructure:"operation"`
}

// MapVariablesToValue :
//						map the input variables to the values.
//						this is not thread safe
func (f *FilterExpression) MapVariablesToValue(vars map[string]interface{}) error {
	if f == nil {
		return nil
	}
	properties := f.Properties
	// base case
	if len(properties) == 0 {
		return nil
	}

	for i := 0; i < len(properties); i++ {
		filter := properties[i]
		if filter.Column != "" && filter.Op != "" && filter.Value == nil && filter.Variable != nil {
			val := vars[*filter.Variable]
			if val == nil {
				return fmt.Errorf("%w : col '%s' : variable '%s' ", errorValueNotFoundForVariable, filter.Column, *filter.Variable)
			}
			filter.Value = val
			filter.Variable = nil
		} else if filter.Properties != nil && len(filter.Properties) > 0 {
			return filter.MapVariablesToValue(vars)
		}
	}
	return nil
}

// FilterExpressionFromMap : generate filter expression from map string interface
func FilterExpressionFromMap(expr map[string]interface{}) (*FilterExpression, error) {
	var f FilterExpression
	err := mapstructure.Decode(expr, &f)
	if err != nil {
		return nil, err
	}
	return &f, nil
}
