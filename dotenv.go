package config

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/joho/godotenv"
)

// maximumDotEnvFileSize bounds what a dotenv file may make the process read at startup.
const maximumDotEnvFileSize = 1 << 20

func dotEnvFiles(options *LoadConfigOptions) []string {
	if options == nil {
		return []string{".env"}
	}
	if options.DisableDotEnv {
		return nil
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
		values, err := readDotEnvFile(file)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for key, value := range values {
			if _, present := staged[key]; !present {
				staged[key] = value
			}
		}
	}
	return staged, nil
}

// readDotEnvFile parses one regular file of at most maximumDotEnvFileSize bytes. Symlinks are
// followed, so a dotenv file mounted from a Kubernetes Secret works. A parse error is reported
// without the parser's message, because that message quotes file content and the file holds secrets.
func readDotEnvFile(file string) (map[string]string, error) {
	information, err := os.Stat(file) // before opening: opening a FIFO would block
	if err != nil {
		return nil, wrapFileError(file, err)
	}
	if !information.Mode().IsRegular() {
		return nil, fmt.Errorf("load dotenv file %s: not a regular file: %w", file, ErrInvalid)
	}
	content, err := readBounded(file)
	if err != nil {
		return nil, err
	}
	values, err := godotenv.UnmarshalBytes(content)
	if err != nil {
		return nil, fmt.Errorf("load dotenv file %s: invalid dotenv syntax (details withheld, the file may hold secrets): %w", file, ErrInvalid)
	}
	return values, nil
}

func readBounded(file string) ([]byte, error) {
	handle, err := os.Open(file) //nolint:gosec // G304: reading the dotenv paths the caller chose is this function's job
	if err != nil {
		return nil, wrapFileError(file, err)
	}
	defer handle.Close()
	content, err := io.ReadAll(io.LimitReader(handle, maximumDotEnvFileSize+1))
	if err != nil {
		return nil, wrapFileError(file, err)
	}
	if len(content) > maximumDotEnvFileSize {
		return nil, fmt.Errorf("load dotenv file %s: larger than %d bytes: %w", file, maximumDotEnvFileSize, ErrInvalid)
	}
	return content, nil
}

func wrapFileError(file string, err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return fmt.Errorf("load dotenv file %s: %w", file, err)
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
