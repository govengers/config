// Package config declares environment-based configuration once, loads and validates all of it at
// startup and binds it into the application's own struct.
//
// A service in a crash loop at startup beats one that crashes when a missing value is first needed:
// LoadConfig fails with every missing or invalid value listed, and the registry getters panic on a
// key that was never declared. Read every value in main, at startup, and both happen before the
// service does anything else.
package config

// Value is a declared configuration value. Create one with String, StringArray, Bool or Int.
type Value interface {
	describe() *declaration
}

// LoadConfigOptions selects the dotenv files LoadConfigWithOptions reads.
type LoadConfigOptions struct {
	// DotEnvFile is read before DotEnvFiles.
	DotEnvFile string
	// DotEnvFiles are read in order. A missing file is skipped; without any file, ".env" is read if it exists.
	DotEnvFiles []string
	// DisableDotEnv reads no dotenv file at all, not even ".env" in the working directory.
	DisableDotEnv bool
}

// LoadConfig loads values from the environment and from a ".env" file in the working directory, if
// there is one. See LoadConfigWithOptions.
func LoadConfig(values []Value) error {
	return LoadConfigWithOptions(values, nil)
}

// LoadConfigWithOptions reads the dotenv files, resolves every value and, when all of them are
// valid, replaces the registry, writes each value into its Into target and sets the dotenv
// variables the process environment does not have yet, so other code in the process sees them too.
// Otherwise it returns an error naming every missing or invalid value, and neither the registry,
// any target nor the process environment changes.
//
// A variable that is already set in the environment wins over the dotenv files, and an earlier
// dotenv file wins over a later one.
func LoadConfigWithOptions(values []Value, options *LoadConfigOptions) error {
	staged, err := readDotEnvFiles(dotEnvFiles(options))
	if err != nil {
		return err
	}
	entries, err := resolveAll(values, environmentWith(staged))
	if err != nil {
		return err
	}
	if err := exportDotEnvValues(staged); err != nil {
		return err
	}
	registry.replace(entries)
	return nil
}
