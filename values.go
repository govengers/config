package config

import "slices"

type valueType string

const (
	stringType      valueType = "string"
	stringArrayType valueType = "string array"
	boolType        valueType = "bool"
	intType         valueType = "int"
)

type declaration struct {
	environmentVariable string
	valueType           valueType
	defaultValue        any // nil: the value is required
	notEmpty            bool
	sensitive           bool
	target              any // *string, *[]string, *bool or *int; nil when not bound
}

// StringValue declares a string environment variable.
type StringValue struct{ declaration declaration }

// String declares the string environment variable environmentVariable. Without a Default it is required.
func String(environmentVariable string) *StringValue {
	return &StringValue{declaration: declaration{environmentVariable: environmentVariable, valueType: stringType}}
}

// Default is used when the variable is not set, which makes the value optional.
func (v *StringValue) Default(value string) *StringValue {
	v.declaration.defaultValue = value
	return v
}

// NotEmpty rejects an empty value, and an empty default.
func (v *StringValue) NotEmpty() *StringValue {
	v.declaration.notEmpty = true
	return v
}

// Sensitive masks the value in Print.
func (v *StringValue) Sensitive() *StringValue {
	v.declaration.sensitive = true
	return v
}

// Into writes the value into target when LoadConfig succeeds, and again on Set and BindPFlag.
func (v *StringValue) Into(target *string) *StringValue {
	v.declaration.target = target
	return v
}

func (v *StringValue) describe() *declaration { return &v.declaration }

// StringArrayValue declares a comma-separated list environment variable.
type StringArrayValue struct{ declaration declaration }

// StringArray declares the list environment variable environmentVariable. Items are split at commas
// and trimmed; empty items are dropped. Without a Default it is required.
func StringArray(environmentVariable string) *StringArrayValue {
	return &StringArrayValue{declaration: declaration{environmentVariable: environmentVariable, valueType: stringArrayType}}
}

// Default is used when the variable is not set, which makes the value optional.
func (v *StringArrayValue) Default(value []string) *StringArrayValue {
	v.declaration.defaultValue = slices.Clone(value)
	if value == nil {
		v.declaration.defaultValue = []string{}
	}
	return v
}

// NotEmpty rejects a list without items, and an empty default.
func (v *StringArrayValue) NotEmpty() *StringArrayValue {
	v.declaration.notEmpty = true
	return v
}

// Sensitive masks the value in Print.
func (v *StringArrayValue) Sensitive() *StringArrayValue {
	v.declaration.sensitive = true
	return v
}

// Into writes the value into target when LoadConfig succeeds, and again on Set and BindPFlag.
func (v *StringArrayValue) Into(target *[]string) *StringArrayValue {
	v.declaration.target = target
	return v
}

func (v *StringArrayValue) describe() *declaration { return &v.declaration }

// BoolValue declares a boolean environment variable that accepts "true" and "false" in any case.
type BoolValue struct{ declaration declaration }

// Bool declares the boolean environment variable environmentVariable. Without a Default it is required.
func Bool(environmentVariable string) *BoolValue {
	return &BoolValue{declaration: declaration{environmentVariable: environmentVariable, valueType: boolType}}
}

// Default is used when the variable is not set, which makes the value optional.
func (v *BoolValue) Default(value bool) *BoolValue {
	v.declaration.defaultValue = value
	return v
}

// Sensitive masks the value in Print.
func (v *BoolValue) Sensitive() *BoolValue {
	v.declaration.sensitive = true
	return v
}

// Into writes the value into target when LoadConfig succeeds, and again on Set and BindPFlag.
func (v *BoolValue) Into(target *bool) *BoolValue {
	v.declaration.target = target
	return v
}

func (v *BoolValue) describe() *declaration { return &v.declaration }

// IntValue declares an integer environment variable.
type IntValue struct{ declaration declaration }

// Int declares the integer environment variable environmentVariable. Without a Default it is required.
func Int(environmentVariable string) *IntValue {
	return &IntValue{declaration: declaration{environmentVariable: environmentVariable, valueType: intType}}
}

// Default is used when the variable is not set, which makes the value optional.
func (v *IntValue) Default(value int) *IntValue {
	v.declaration.defaultValue = value
	return v
}

// Sensitive masks the value in Print.
func (v *IntValue) Sensitive() *IntValue {
	v.declaration.sensitive = true
	return v
}

// Into writes the value into target when LoadConfig succeeds, and again on Set and BindPFlag.
func (v *IntValue) Into(target *int) *IntValue {
	v.declaration.target = target
	return v
}

func (v *IntValue) describe() *declaration { return &v.declaration }
