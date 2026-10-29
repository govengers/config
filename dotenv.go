package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/joho/godotenv"
)

func dotEnvFiles(options *LoadConfigOptions) []string {
	if options == nil {
		return []string{".env"}
	}
	files := []string{}
	if options.DotEnvFile != "" {
		files = append(files, options.DotEnvFile)
	}
	files = append(files, options.DotEnvFiles...)
	if len(files) == 0 {
		return []string{".env"}
	}
	return files
}

// readDotEnvFiles parses every existing file without touching the process environment. Missing
// files are skipped, a file that exists but cannot be read or parsed is an error, and for a key in
// several files the earliest file wins.
func readDotEnvFiles(files []string) (map[string]string, error) {
	staged := map[string]string{}
	for _, file := range files {
		values, err := godotenv.Read(file)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("load dotenv file %s: %w", file, err)
		}
		for key, value := range values {
			if _, present := staged[key]; !present {
				staged[key] = value
			}
		}
	}
	return staged, nil
}

// environmentWith looks a key up in the process environment first and in the staged dotenv values second.
func environmentWith(staged map[string]string) func(key string) (string, bool) {
	return func(key string) (string, bool) {
		if value, present := os.LookupEnv(key); present {
			return value, true
		}
		value, present := staged[key]
		return value, present
	}
}

// exportDotEnvValues sets every staged value the process environment does not have yet.
func exportDotEnvValues(staged map[string]string) error {
	for key, value := range staged {
		if _, present := os.LookupEnv(key); present {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set %s from a dotenv file: %w", key, err)
		}
	}
	return nil
}
