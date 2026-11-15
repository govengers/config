package config

import (
	"errors"
	"fmt"
	"math"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
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
	if err == nil {
		err = checkValue(current, value)
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", current.environmentVariable, describeParseError(current, raw, err))
	}
	if current.notEmpty && isEmpty(value) {
		return nil, fmt.Errorf("%s must not be empty: %w", current.environmentVariable, ErrInvalid)
	}
	return &entry{declaration: current, value: value, provided: true}, nil
}

// checkDeclaration rejects declarations that contradict themselves, whatever the environment holds.
func checkDeclaration(current declaration) error {
	if current.valueType == enumType && len(current.allowed) == 0 {
		return fmt.Errorf("%s is an Enum without allowed values: %w", current.environmentVariable, ErrInvalid)
	}
	if hasNilTarget(current.target) {
		return fmt.Errorf("%s is bound with Into to a nil pointer: %w", current.environmentVariable, ErrInvalid)
	}
	if current.defaultValue == nil {
		return nil
	}
	value, err := typedDefault(current)
	if err == nil {
		err = checkValue(current, value)
	}
	if err != nil {
		return fmt.Errorf("%s has an invalid default: %w", current.environmentVariable, describeParseError(current, fmt.Sprint(current.defaultValue), err))
	}
	if current.notEmpty && isEmpty(value) {
		return fmt.Errorf("%s is NotEmpty but its default is empty: %w", current.environmentVariable, ErrInvalid)
	}
	return nil
}

func resolveDefault(current declaration) (*entry, error) {
	if current.defaultValue == nil {
		return nil, fmt.Errorf("%s is not set and has no default: %w", current.environmentVariable, ErrMissing)
	}
	value, err := typedDefault(current)
	if err != nil {
		return nil, err // unreachable: checkDeclaration parsed the default already
	}
	return &entry{declaration: current, value: value}, nil
}

// typedDefault returns the default as the value's Go type. A URL default is declared as text, so
// it reads like the environment variable, and is parsed here.
func typedDefault(current declaration) (any, error) {
	if current.valueType == urlType {
		return parseURL(current.defaultValue.(string))
	}
	return copyValue(current.defaultValue), nil
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
	case *time.Duration:
		return typed == nil
	case *float64:
		return typed == nil
	case *url.URL:
		return typed == nil
	default:
		return false
	}
}

func parse(kind valueType, raw string) (any, error) {
	switch kind {
	case stringType, enumType:
		return raw, nil
	case stringArrayType:
		return parseStringArray(raw), nil
	case boolType:
		value, err := parseBool(raw)
		return value, err
	case intType:
		value, err := parseInt(raw)
		return value, err
	case durationType:
		value, err := parseDuration(raw)
		return value, err
	case floatType:
		value, err := parseFloat(raw)
		return value, err
	case urlType:
		value, err := parseURL(raw)
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
		return false, fmt.Errorf("is neither true nor false: %w", ErrInvalid)
	}
}

func parseInt(raw string) (int, error) {
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("is not an integer: %w", ErrInvalid)
	}
	return value, nil
}

func parseDuration(raw string) (time.Duration, error) {
	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("is not a duration such as 30s, 5m or 1h30m: %w", ErrInvalid)
	}
	return value, nil
}

func parseFloat(raw string) (float64, error) {
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("is not a number: %w", ErrInvalid)
	}
	return value, nil
}

// parseURL drops the error of url.Parse, because it quotes the value, which may hold credentials.
func parseURL(raw string) (url.URL, error) {
	if raw == "" {
		return url.URL{}, nil
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Hostname() == "" {
		return url.URL{}, fmt.Errorf("is not an absolute URL with scheme and host: %w", ErrInvalid)
	}
	return *parsed, nil
}

// checkValue applies the rules a declaration adds to its type: Enum members, URL schemes and
// finite floats. It runs on values from the environment, defaults, flags and Set alike.
func checkValue(current declaration, value any) error {
	switch typed := value.(type) {
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) {
			return fmt.Errorf("is not a finite number: %w", ErrInvalid)
		}
	case url.URL:
		if len(current.allowed) > 0 && typed != (url.URL{}) && !slices.Contains(current.allowed, typed.Scheme) {
			return fmt.Errorf("has a scheme other than %s: %w", strings.Join(current.allowed, ", "), ErrInvalid)
		}
	case string:
		if current.valueType == enumType && !slices.Contains(current.allowed, typed) {
			return fmt.Errorf("is not one of %s: %w", strings.Join(current.allowed, ", "), ErrInvalid)
		}
	}
	return nil
}

// describeParseError names the rejected value only when it is not Sensitive, so a malformed secret
// never reaches the startup error and the logs that collect it. A URL is quoted redacted, like Print
// shows it, and not at all when its password cannot be located: url.Parse fails, or the text holds
// an "@" that url.Parse did not read as user info (a missing scheme or "//" leaves it in the path).
func describeParseError(current declaration, raw string, err error) error {
	if current.sensitive {
		return fmt.Errorf("the value (sensitive, not shown) %w", err)
	}
	if current.valueType == urlType {
		parsed, parseErr := url.Parse(raw)
		if parseErr != nil || (parsed.User == nil && strings.Contains(raw, "@")) {
			return fmt.Errorf("the value (unparsable, not shown) %w", err)
		}
		raw = parsed.Redacted()
	}
	return fmt.Errorf("%q %w", raw, err)
}

// isEmpty treats a value of only whitespace as empty: it is always a mistake for a NotEmpty value.
func isEmpty(value any) bool {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed) == ""
	case []string:
		return len(typed) == 0
	case url.URL:
		return typed == url.URL{}
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
	case *time.Duration:
		*target = e.value.(time.Duration)
	case *float64:
		*target = e.value.(float64)
	case *url.URL:
		*target = e.value.(url.URL)
	}
}
