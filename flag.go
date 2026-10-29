package config

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
)

// BindPFlag overrides the value of key with flag if the flag was set on the command line, and writes
// it into the bound field. An unset flag keeps the environment value. Call it after LoadConfig.
func BindPFlag(key string, flag *pflag.Flag) error {
	if flag == nil || !flag.Changed {
		return nil
	}
	current, err := registry.declarationOf(key)
	if err != nil {
		return err
	}
	value, err := parseFlag(current, flag)
	if err != nil {
		return fmt.Errorf("flag --%s for %s: %w", flag.Name, key, err)
	}
	return registry.update(key, current.valueType, value, true)
}

func parseFlag(current declaration, flag *pflag.Flag) (any, error) {
	if sliceValue, isSlice := flag.Value.(pflag.SliceValue); isSlice && current.valueType == stringArrayType {
		return parseStringArray(strings.Join(sliceValue.GetSlice(), ",")), nil
	}
	raw := flag.Value.String()
	value, err := parse(current.valueType, raw)
	if err != nil {
		return nil, describeParseError(current, raw, err)
	}
	return value, nil
}
