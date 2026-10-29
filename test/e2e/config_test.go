//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// appBinary is the example program, built once by TestMain.
var appBinary string

func TestMain(m *testing.M) {
	directory, err := os.MkdirTemp("", "config-e2e")
	if err != nil {
		panic(err)
	}
	appBinary = filepath.Join(directory, "app")
	if err := buildApp(appBinary); err != nil {
		panic(err)
	}
	code := m.Run()
	_ = os.RemoveAll(directory) // temporary build output
	os.Exit(code)
}

func buildApp(output string) error {
	buildContext, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	build := exec.CommandContext(buildContext, "go", "build", "-o", output, "./testdata/app")
	build.Stdout, build.Stderr = os.Stdout, os.Stderr
	return build.Run()
}

type result struct {
	exitCode      int
	stdout        string
	stderr        string
	configuration configuration
}

type configuration struct {
	ServiceName    string
	Port           int
	DevMode        bool
	DatabaseURL    string
	AllowedOrigins []string
}

// runApp starts the app in workingDirectory with only the given environment variables (plus PATH),
// so nothing from the developer's shell leaks in.
func runApp(t *testing.T, workingDirectory string, environment map[string]string, arguments ...string) result {
	t.Helper()
	command := exec.CommandContext(t.Context(), appBinary, arguments...)
	command.Dir = workingDirectory
	command.Env = []string{"PATH=" + os.Getenv("PATH")}
	for key, value := range environment {
		command.Env = append(command.Env, key+"="+value)
	}
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr

	err := command.Run()

	outcome := result{stdout: stdout.String(), stderr: stderr.String()}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		outcome.exitCode = exitError.ExitCode()
		return outcome
	}
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &outcome.configuration))
	return outcome
}

func TestStartsWithEveryValueBoundIntoTheStruct(t *testing.T) {
	outcome := runApp(t, t.TempDir(), map[string]string{
		"SERVICE_NAME":    "billing",
		"PORT":            "9000",
		"DEV_MODE":        "true",
		"DATABASE_URL":    "postgres://billing:secret@db/billing",
		"ALLOWED_ORIGINS": "https://a.example, https://b.example",
	})

	require.Equal(t, 0, outcome.exitCode, outcome.stderr)
	require.Equal(t, configuration{
		ServiceName:    "billing",
		Port:           9000,
		DevMode:        true,
		DatabaseURL:    "postgres://billing:secret@db/billing",
		AllowedOrigins: []string{"https://a.example", "https://b.example"},
	}, outcome.configuration)
}

func TestDefaultsFillEverythingOptional(t *testing.T) {
	outcome := runApp(t, t.TempDir(), map[string]string{"DATABASE_URL": "postgres://db"})

	require.Equal(t, 0, outcome.exitCode, outcome.stderr)
	require.Equal(t, configuration{
		ServiceName:    "e2e-app",
		Port:           8080,
		DatabaseURL:    "postgres://db",
		AllowedOrigins: []string{"http://localhost"},
	}, outcome.configuration)
}

func TestFailsAtStartupListingEveryProblem(t *testing.T) {
	outcome := runApp(t, t.TempDir(), map[string]string{"PORT": "eighty", "DEV_MODE": "yes"})

	require.Equal(t, 1, outcome.exitCode)
	require.Contains(t, outcome.stderr, "DATABASE_URL is not set and has no default")
	require.Contains(t, outcome.stderr, `PORT: "eighty" is not an integer`)
	require.Contains(t, outcome.stderr, `DEV_MODE: "yes" is neither true nor false`)
	require.Empty(t, outcome.stdout)
}

func TestRejectsAnEmptyRequiredValue(t *testing.T) {
	outcome := runApp(t, t.TempDir(), map[string]string{"DATABASE_URL": ""})

	require.Equal(t, 1, outcome.exitCode)
	require.Contains(t, outcome.stderr, "DATABASE_URL must not be empty")
}

func TestReadingAnUndeclaredKeyCrashesAtStartup(t *testing.T) {
	outcome := runApp(t, t.TempDir(), map[string]string{"DATABASE_URL": "postgres://db"}, "--read-undeclared")

	require.Equal(t, 2, outcome.exitCode) // an unrecovered panic exits with 2
	require.Contains(t, outcome.stderr, "panic: config: NOT_DECLARED was never declared")
}

func TestMasksSensitiveValuesInTheTable(t *testing.T) {
	outcome := runApp(t, t.TempDir(), map[string]string{"DATABASE_URL": "postgres://billing:secret@db/billing"})

	require.Equal(t, 0, outcome.exitCode, outcome.stderr)
	require.Contains(t, outcome.stderr, "Application Configuration")
	require.Contains(t, outcome.stderr, "DATABASE_URL")
	require.NotContains(t, outcome.stderr, "secret")
}

func TestReadsDotEnvFromTheWorkingDirectory(t *testing.T) {
	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, ".env"), []byte("DATABASE_URL=postgres://from-dotenv\nPORT=7000\n"), 0o600))

	outcome := runApp(t, directory, map[string]string{"PORT": "7100"})

	require.Equal(t, 0, outcome.exitCode, outcome.stderr)
	require.Equal(t, "postgres://from-dotenv", outcome.configuration.DatabaseURL)
	require.Equal(t, 7100, outcome.configuration.Port, "the environment wins over the dotenv file")
}

func TestReadsExplicitDotEnvFilesAndSkipsMissingOnes(t *testing.T) {
	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "local.env"), []byte("DATABASE_URL=postgres://local\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "shared.env"), []byte("DATABASE_URL=postgres://shared\nSERVICE_NAME=shared\n"), 0o600))

	outcome := runApp(t, directory, nil, "--dotenv-file=missing.env,local.env,shared.env")

	require.Equal(t, 0, outcome.exitCode, outcome.stderr)
	require.Equal(t, "postgres://local", outcome.configuration.DatabaseURL, "the first file wins")
	require.Equal(t, "shared", outcome.configuration.ServiceName)
}

func TestAFlagOverridesTheEnvironmentOnlyWhenSet(t *testing.T) {
	environment := map[string]string{"DATABASE_URL": "postgres://db", "PORT": "9000"}

	withFlag := runApp(t, t.TempDir(), environment, "--port=9100")
	withoutFlag := runApp(t, t.TempDir(), environment)

	require.Equal(t, 9100, withFlag.configuration.Port)
	require.Equal(t, 9000, withoutFlag.configuration.Port)
}

func TestADotEnvSyntaxErrorKeepsTheFileContentOutOfStderr(t *testing.T) {
	directory := t.TempDir()
	content := "DATABASE_URL=postgres://billing:dummy-secret@db/billing\nTLS_KEY=-----BEGIN PRIVATE KEY-----\nMIIBdummybody\n"
	require.NoError(t, os.WriteFile(filepath.Join(directory, "broken.env"), []byte(content), 0o600))

	outcome := runApp(t, directory, nil, "--dotenv-file=broken.env")

	require.Equal(t, 1, outcome.exitCode)
	require.Contains(t, outcome.stderr, "load dotenv file broken.env: invalid dotenv syntax")
	require.NotContains(t, outcome.stderr, "dummy-secret")
	require.NotContains(t, outcome.stderr, "MIIBdummybody")
}
