package config

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// ErrMissing reports a required value whose environment variable is not set.
var ErrMissing = errors.New("missing")

// ErrInvalid reports a value that cannot be parsed or breaks its declaration.
var ErrInvalid = errors.New("invalid")

type entry struct {
	declaration declaration
	value       any
	provided    bool
}

// resolveAll resolves every value and returns all problems at once, so one failed start shows the
// whole list of what to fix.
func resolveAll(values []Value, lookup func(key string) (string, bool)) (map[string]*entry, error) {
	entries := make(map[string]*entry, len(values))
	declared := make(map[string]bool, len(values))
	var problems []error
	for _, value := range values {
		current := *value.describe()
		if declared[current.environmentVariable] {
			problems = append(problems, fmt.Errorf("%s is declared twice: %w", current.environmentVariable, ErrInvalid))
			continue
		}
		declared[current.environmentVariable] = true
		resolved, err := resolve(current, lookup)
		if err != nil {
			problems = append(problems, err)
			continue
		}
		entries[current.environmentVariable] = resolved
	}
	if len(problems) > 0 {
		return nil, errors.Join(problems...)
	}
	return entries, nil
}

func resolve(current declaration, lookup func(key string) (string, bool)) (*entry, error) {
	if err := checkDeclaration(current); err != nil {
		return nil, err
	}
	raw, provided := lookup(current.environmentVariable)
	if !provided {
		return resolveDefault(current)
	}
	value, err := parse(current.valueType, raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", current.environmentVariable, err)
	}
	if current.notEmpty && isEmpty(value) {
		return nil, fmt.Errorf("%s must not be empty: %w", current.environmentVariable, ErrInvalid)
	}
	return &entry{declaration: current, value: value, provided: true}, nil
}

// checkDeclaration rejects declarations that contradict themselves, whatever the environment holds.
func checkDeclaration(current declaration) error {
	if current.notEmpty && current.defaultValue != nil && isEmpty(current.defaultValue) {
		return fmt.Errorf("%s is NotEmpty but its default is empty: %w", current.environmentVariable, ErrInvalid)
	}
	if hasNilTarget(current.target) {
		return fmt.Errorf("%s is bound with Into to a nil pointer: %w", current.environmentVariable, ErrInvalid)
	}
	return nil
}

func resolveDefault(current declaration) (*entry, error) {
	if current.defaultValue == nil {
		return nil, fmt.Errorf("%s is not set and has no default: %w", current.environmentVariable, ErrMissing)
	}
	return &entry{declaration: current, value: copyValue(current.defaultValue)}, nil
}

func hasNilTarget(target any) bool {
	switch typed := target.(type) {
	case *string:
		return typed == nil
	case *[]string:
		return typed == nil
	case *bool:
		return typed == nil
	case *int:
		return typed == nil
	default:
		return false
	}
}

func parse(kind valueType, raw string) (any, error) {
	switch kind {
	case stringType:
		return raw, nil
	case stringArrayType:
		return parseStringArray(raw), nil
	case boolType:
		value, err := parseBool(raw)
		return value, err
	case intType:
		value, err := parseInt(raw)
		return value, err
	default:
		return nil, fmt.Errorf("unknown value type %q: %w", kind, ErrInvalid)
	}
}

func parseStringArray(raw string) []string {
	items := []string{}
	for item := range strings.SplitSeq(raw, ",") {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}

func parseBool(raw string) (bool, error) {
	switch strings.ToLower(raw) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("%q is neither true nor false: %w", raw, ErrInvalid)
	}
}

func parseInt(raw string) (int, error) {
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%q is not an integer: %w", raw, ErrInvalid)
	}
	return value, nil
}

func isEmpty(value any) bool {
	switch typed := value.(type) {
	case string:
		return typed == ""
	case []string:
		return len(typed) == 0
	default:
		return false
	}
}

func copyValue(value any) any {
	if items, isList := value.([]string); isList {
		return slices.Clone(items)
	}
	return value
}

// writeTarget copies the value into the field bound with Into, if there is one.
func (e *entry) writeTarget() {
	switch target := e.declaration.target.(type) {
	case *string:
		*target = e.value.(string)
	case *[]string:
		*target = slices.Clone(e.value.([]string))
	case *bool:
		*target = e.value.(bool)
	case *int:
		*target = e.value.(int)
	}
}
