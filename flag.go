package config

import (
	"fmt"
	"slices"

	"github.com/spf13/pflag"
)

// BindPFlag overrides the value of key with flag if the flag was set on the command line, and writes
// it into the bound field. An unset flag keeps the environment value. Call it after LoadConfig.
func BindPFlag(key string, flag *pflag.Flag) error {
	if flag == nil || !flag.Changed {
		return nil
	}
	expected, err := registry.typeOf(key)
	if err != nil {
		return err
	}
	value, err := parseFlag(expected, flag)
	if err != nil {
		return fmt.Errorf("flag --%s for %s: %w", flag.Name, key, err)
	}
	return registry.update(key, expected, value, true)
}

func parseFlag(expected valueType, flag *pflag.Flag) (any, error) {
	if sliceValue, isSlice := flag.Value.(pflag.SliceValue); isSlice && expected == stringArrayType {
		return slices.Clone(sliceValue.GetSlice()), nil
	}
	return parse(expected, flag.Value.String())
}
