package config

import (
	"errors"
	"fmt"
	"io/fs"

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

// loadDotEnvFiles sets the variables of every existing file that are not set yet. Missing files are
// skipped; a file that exists but cannot be read or parsed is an error.
func loadDotEnvFiles(files []string) error {
	for _, file := range files {
		err := godotenv.Load(file)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("load dotenv file %s: %w", file, err)
		}
	}
	return nil
}
