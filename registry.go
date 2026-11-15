package config

import (
	"fmt"
	"net/url"
	"slices"
	"sync"
	"time"
)

// Registry holds the values LoadConfig loaded. It is safe for concurrent use.
type Registry struct {
	mutex   sync.RWMutex
	entries map[string]*entry
}

// registry is process-wide by design: Get and Set work from anywhere in main without passing it
// around. It is the declared DES-003 exception of this library (.govengers.yml).
var registry = &Registry{entries: map[string]*entry{}}

// Get returns the registry LoadConfig filled. Its getters panic on a key that was never declared or
// that was declared with another type, so a typo crashes the first time it runs; read every value in
// main at startup and that is before the service does anything.
func Get() *Registry {
	return registry
}

// String returns the string value of key.
func (r *Registry) String(key string) string {
	return read[string](r, key, stringType)
}

// StringArray returns a copy of the list value of key.
func (r *Registry) StringArray(key string) []string {
	return slices.Clone(read[[]string](r, key, stringArrayType))
}

// Bool returns the boolean value of key.
func (r *Registry) Bool(key string) bool {
	return read[bool](r, key, boolType)
}

// Int returns the integer value of key.
func (r *Registry) Int(key string) int {
	return read[int](r, key, intType)
}

// Duration returns the duration value of key.
func (r *Registry) Duration(key string) time.Duration {
	return read[time.Duration](r, key, durationType)
}

// Float returns the floating-point value of key.
func (r *Registry) Float(key string) float64 {
	return read[float64](r, key, floatType)
}

// URL returns a fresh copy of the URL value of key, never nil; a zero url.URL when the value is
// empty. A pointer, because every url.URL method has a pointer receiver.
func (r *Registry) URL(key string) *url.URL {
	value := read[url.URL](r, key, urlType)
	return &value
}

// Enum returns the value of key, one of its allowed values.
func (r *Registry) Enum(key string) string {
	return read[string](r, key, enumType)
}

func read[T any](source *Registry, key string, expected valueType) T {
	source.mutex.RLock()
	defer source.mutex.RUnlock()
	found, err := source.find(key, expected)
	if err != nil {
		panic(err)
	}
	return found.value.(T)
}

// find returns the entry of key if it was declared with the expected type. Callers hold the lock.
func (r *Registry) find(key string, expected valueType) (*entry, error) {
	found, declared := r.entries[key]
	if !declared {
		return nil, fmt.Errorf("config: %s was never declared", key)
	}
	if found.declaration.valueType != expected {
		return nil, fmt.Errorf("config: %s is declared as %s, not as %s", key, found.declaration.valueType, expected)
	}
	return found, nil
}

// check reports whether key was declared with the expected type, so a Setter that parses text
// names a wrong key or type before it blames the text.
func (r *Registry) check(key string, expected valueType) error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	_, err := r.find(key, expected)
	return err
}

func (r *Registry) declarationOf(key string) (declaration, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	found, declared := r.entries[key]
	if !declared {
		return declaration{}, fmt.Errorf("config: %s was never declared", key)
	}
	return found.declaration, nil
}

// replace swaps in the loaded entries and writes every Into target, under one lock.
func (r *Registry) replace(entries map[string]*entry) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.entries = entries
	for _, loaded := range entries {
		loaded.writeTarget()
	}
}

// update sets the value of key and writes it into the bound field. It applies the same NotEmpty,
// Enum, scheme and finite-number rules as LoadConfig, so no path can store a value LoadConfig rejects.
func (r *Registry) update(key string, expected valueType, value any, provided bool) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	found, err := r.find(key, expected)
	if err != nil {
		return err
	}
	if found.declaration.notEmpty && isEmpty(value) {
		return fmt.Errorf("config: %s must not be empty: %w", key, ErrInvalid)
	}
	if err := checkValue(found.declaration, value); err != nil {
		return fmt.Errorf("config: %s %w", key, err)
	}
	found.value = value
	found.provided = found.provided || provided
	found.writeTarget()
	return nil
}

// Setter changes loaded values, for tests. Set returns it.
type Setter struct{}

// Set returns the Setter. Its methods panic on a key that was never declared, that was declared
// with another type, or on a value LoadConfig would reject.
func Set() *Setter {
	return &Setter{}
}

// String sets the string value of key.
func (s *Setter) String(key string, value string) {
	mustUpdate(key, stringType, value)
}

// StringArray sets the list value of key from a comma-separated string, parsed like the environment variable.
func (s *Setter) StringArray(key string, value string) {
	mustUpdate(key, stringArrayType, parseStringArray(value))
}

// Bool sets the boolean value of key.
func (s *Setter) Bool(key string, value bool) {
	mustUpdate(key, boolType, value)
}

// Int sets the integer value of key.
func (s *Setter) Int(key string, value int) {
	mustUpdate(key, intType, value)
}

// Duration sets the duration value of key.
func (s *Setter) Duration(key string, value time.Duration) {
	mustUpdate(key, durationType, value)
}

// Float sets the floating-point value of key.
func (s *Setter) Float(key string, value float64) {
	mustUpdate(key, floatType, value)
}

// URL sets the URL value of key from text, parsed like the environment variable.
func (s *Setter) URL(key string, value string) {
	if err := registry.check(key, urlType); err != nil {
		panic(err)
	}
	parsed, err := parseURL(value)
	if err != nil {
		panic(fmt.Errorf("config: %s %w", key, err))
	}
	mustUpdate(key, urlType, parsed)
}

// Enum sets the value of key, which must be one of its allowed values.
func (s *Setter) Enum(key string, value string) {
	mustUpdate(key, enumType, value)
}

func mustUpdate(key string, expected valueType, value any) {
	if err := registry.update(key, expected, value, false); err != nil {
		panic(err)
	}
}
