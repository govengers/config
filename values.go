package config

import (
	"net/url"
	"slices"
	"strings"
	"time"
)

type valueType string

const (
	stringType      valueType = "string"
	stringArrayType valueType = "string array"
	boolType        valueType = "bool"
	intType         valueType = "int"
	durationType    valueType = "duration"
	floatType       valueType = "float"
	urlType         valueType = "url"
	enumType        valueType = "enum"
)

type declaration struct {
	environmentVariable string
	valueType           valueType
	defaultValue        any // nil: the value is required; a URL default is the text Default was given
	notEmpty            bool
	sensitive           bool
	allowed             []string // the members of an Enum, or the schemes of a URL; empty for a URL accepts any scheme
	target              any      // a pointer to the value's Go type; nil when not bound
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

// DurationValue declares a duration environment variable in Go syntax, such as "30s", "5m" or "1h30m".
type DurationValue struct{ declaration declaration }

// Duration declares the duration environment variable environmentVariable. A number without a unit
// is rejected, except "0", so "300" never silently means 300 nanoseconds. Without a Default it is
// required.
func Duration(environmentVariable string) *DurationValue {
	return &DurationValue{declaration: declaration{environmentVariable: environmentVariable, valueType: durationType}}
}

// Default is used when the variable is not set, which makes the value optional.
func (v *DurationValue) Default(value time.Duration) *DurationValue {
	v.declaration.defaultValue = value
	return v
}

// Sensitive masks the value in Print.
func (v *DurationValue) Sensitive() *DurationValue {
	v.declaration.sensitive = true
	return v
}

// Into writes the value into target when LoadConfig succeeds, and again on Set and BindPFlag.
func (v *DurationValue) Into(target *time.Duration) *DurationValue {
	v.declaration.target = target
	return v
}

func (v *DurationValue) describe() *declaration { return &v.declaration }

// FloatValue declares a floating-point environment variable. NaN and infinities are rejected.
type FloatValue struct{ declaration declaration }

// Float declares the floating-point environment variable environmentVariable. Without a Default it is required.
func Float(environmentVariable string) *FloatValue {
	return &FloatValue{declaration: declaration{environmentVariable: environmentVariable, valueType: floatType}}
}

// Default is used when the variable is not set, which makes the value optional.
func (v *FloatValue) Default(value float64) *FloatValue {
	v.declaration.defaultValue = value
	return v
}

// Sensitive masks the value in Print.
func (v *FloatValue) Sensitive() *FloatValue {
	v.declaration.sensitive = true
	return v
}

// Into writes the value into target when LoadConfig succeeds, and again on Set and BindPFlag.
func (v *FloatValue) Into(target *float64) *FloatValue {
	v.declaration.target = target
	return v
}

func (v *FloatValue) describe() *declaration { return &v.declaration }

// URLValue declares a URL environment variable. A non-empty value must be an absolute URL with a
// scheme and a host; an empty value is the zero url.URL unless NotEmpty rejects it.
type URLValue struct{ declaration declaration }

// URL declares the URL environment variable environmentVariable. Without a Default it is required.
func URL(environmentVariable string) *URLValue {
	return &URLValue{declaration: declaration{environmentVariable: environmentVariable, valueType: urlType}}
}

// Default is parsed like the environment variable and used when the variable is not set, which
// makes the value optional. An invalid default fails LoadConfig.
func (v *URLValue) Default(value string) *URLValue {
	v.declaration.defaultValue = value
	return v
}

// NotEmpty rejects an empty value, and an empty default.
func (v *URLValue) NotEmpty() *URLValue {
	v.declaration.notEmpty = true
	return v
}

// Schemes accepts only URLs whose scheme is one of schemes, compared without case. Without it any
// scheme is accepted.
func (v *URLValue) Schemes(schemes ...string) *URLValue {
	v.declaration.allowed = make([]string, 0, len(schemes))
	for _, scheme := range schemes {
		v.declaration.allowed = append(v.declaration.allowed, strings.ToLower(scheme))
	}
	return v
}

// Sensitive masks the value in Print. Declare a URL with credentials Sensitive: Print hides only
// the password of a URL that is not.
func (v *URLValue) Sensitive() *URLValue {
	v.declaration.sensitive = true
	return v
}

// Into writes the value into target when LoadConfig succeeds, and again on Set and BindPFlag.
func (v *URLValue) Into(target *url.URL) *URLValue {
	v.declaration.target = target
	return v
}

func (v *URLValue) describe() *declaration { return &v.declaration }

// EnumValue declares a string environment variable restricted to a closed set of values.
type EnumValue struct{ declaration declaration }

// Enum declares the string environment variable environmentVariable, which must be exactly one of
// allowed, compared with case. Without a Default it is required.
func Enum(environmentVariable string, allowed ...string) *EnumValue {
	return &EnumValue{declaration: declaration{environmentVariable: environmentVariable, valueType: enumType, allowed: slices.Clone(allowed)}}
}

// Default is used when the variable is not set, which makes the value optional. A default outside
// the allowed values fails LoadConfig.
func (v *EnumValue) Default(value string) *EnumValue {
	v.declaration.defaultValue = value
	return v
}

// Sensitive masks the value in Print.
func (v *EnumValue) Sensitive() *EnumValue {
	v.declaration.sensitive = true
	return v
}

// Into writes the value into target when LoadConfig succeeds, and again on Set and BindPFlag.
func (v *EnumValue) Into(target *string) *EnumValue {
	v.declaration.target = target
	return v
}

func (v *EnumValue) describe() *declaration { return &v.declaration }
