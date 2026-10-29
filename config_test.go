package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type testConfiguration struct {
	Name    string
	Origins []string
	Debug   bool
	Port    int
}

func testValues(configuration *testConfiguration) []Value {
	return []Value{
		String("CONFIG_TEST_NAME").Default("default-name").Into(&configuration.Name),
		StringArray("CONFIG_TEST_ORIGINS").Default([]string{"https://default.example"}).Into(&configuration.Origins),
		Bool("CONFIG_TEST_DEBUG").Default(false).Into(&configuration.Debug),
		Int("CONFIG_TEST_PORT").Default(8080).Into(&configuration.Port),
	}
}

func TestIntoBindsEveryType(t *testing.T) {
	t.Setenv("CONFIG_TEST_NAME", "service")
	t.Setenv("CONFIG_TEST_ORIGINS", "https://a.example,https://b.example")
	t.Setenv("CONFIG_TEST_DEBUG", "true")
	t.Setenv("CONFIG_TEST_PORT", "9090")
	var configuration testConfiguration

	require.NoError(t, LoadConfig(testValues(&configuration)))

	require.Equal(t, testConfiguration{
		Name:    "service",
		Origins: []string{"https://a.example", "https://b.example"},
		Debug:   true,
		Port:    9090,
	}, configuration)
}

func TestAFailedLoadChangesNeitherTheTargetsNorTheRegistry(t *testing.T) {
	var configuration testConfiguration
	require.NoError(t, LoadConfig(testValues(&configuration)))
	t.Setenv("CONFIG_TEST_NAME", "changed")
	t.Setenv("CONFIG_TEST_PORT", "not-a-number")

	err := LoadConfig(testValues(&configuration))

	require.ErrorIs(t, err, ErrInvalid)
	require.Equal(t, "default-name", configuration.Name)
	require.Equal(t, "default-name", Get().String("CONFIG_TEST_NAME"))
}

func TestDeclarationsCanBeLoadedAgain(t *testing.T) {
	var configuration testConfiguration
	values := testValues(&configuration)
	require.NoError(t, LoadConfig(values))

	t.Setenv("CONFIG_TEST_PORT", "7070")
	require.NoError(t, LoadConfig(values))

	require.Equal(t, 7070, configuration.Port)
	require.Equal(t, 7070, Get().Int("CONFIG_TEST_PORT"))
}

func TestListsAreNotSharedWithTheCaller(t *testing.T) {
	defaults := []string{"https://default.example"}
	var origins []string
	require.NoError(t, LoadConfig([]Value{StringArray("CONFIG_TEST_UNSET_ORIGINS").Default(defaults).Into(&origins)}))

	defaults[0] = "changed by the caller"
	origins[0] = "changed through the target"
	read := Get().StringArray("CONFIG_TEST_UNSET_ORIGINS")
	read[0] = "changed through the getter"

	require.Equal(t, []string{"https://default.example"}, Get().StringArray("CONFIG_TEST_UNSET_ORIGINS"))
}

func TestGetPanicsOnAKeyThatWasNeverDeclared(t *testing.T) {
	require.NoError(t, LoadConfig([]Value{String("CONFIG_TEST_NAME").Default("x")}))

	require.PanicsWithError(t, "config: CONFIG_TEST_TYPO was never declared", func() { Get().String("CONFIG_TEST_TYPO") })
	require.Panics(t, func() { Get().StringArray("CONFIG_TEST_TYPO") })
	require.Panics(t, func() { Get().Bool("CONFIG_TEST_TYPO") })
	require.Panics(t, func() { Get().Int("CONFIG_TEST_TYPO") })
}

func TestGetPanicsOnTheWrongType(t *testing.T) {
	require.NoError(t, LoadConfig([]Value{String("CONFIG_TEST_NAME").Default("x")}))

	require.PanicsWithError(t, "config: CONFIG_TEST_NAME is declared as string, not as int", func() { Get().Int("CONFIG_TEST_NAME") })
}

func TestSetUpdatesTheValueAndTheBoundField(t *testing.T) {
	var configuration testConfiguration
	require.NoError(t, LoadConfig(testValues(&configuration)))

	Set().String("CONFIG_TEST_NAME", "set")
	Set().StringArray("CONFIG_TEST_ORIGINS", "https://x.example, https://y.example")
	Set().Bool("CONFIG_TEST_DEBUG", true)
	Set().Int("CONFIG_TEST_PORT", 1234)

	require.Equal(t, testConfiguration{
		Name:    "set",
		Origins: []string{"https://x.example", "https://y.example"},
		Debug:   true,
		Port:    1234,
	}, configuration)
	require.Equal(t, "set", Get().String("CONFIG_TEST_NAME"))
	require.Equal(t, 1234, Get().Int("CONFIG_TEST_PORT"))
}

func TestSetPanicsOnAnUndeclaredKeyOrTheWrongType(t *testing.T) {
	require.NoError(t, LoadConfig([]Value{String("CONFIG_TEST_NAME").Default("x")}))

	require.Panics(t, func() { Set().String("CONFIG_TEST_TYPO", "x") })
	require.Panics(t, func() { Set().Bool("CONFIG_TEST_NAME", true) })
}

func TestConcurrentReadsAndWrites(t *testing.T) {
	require.NoError(t, LoadConfig([]Value{Int("CONFIG_TEST_COUNTER").Default(0)}))

	var waitGroup sync.WaitGroup
	for index := range 8 {
		waitGroup.Go(func() { Set().Int("CONFIG_TEST_COUNTER", index) })
		waitGroup.Go(func() { _ = Get().Int("CONFIG_TEST_COUNTER") })
	}
	waitGroup.Wait()

	require.GreaterOrEqual(t, Get().Int("CONFIG_TEST_COUNTER"), 0)
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

func TestLoadReadsDotEnvFromTheWorkingDirectory(t *testing.T) {
	directory := t.TempDir()
	t.Chdir(directory)
	writeFile(t, filepath.Join(directory, ".env"), "CONFIG_TEST_DOTENV_NAME=from-dotenv\n")
	t.Cleanup(func() { _ = os.Unsetenv("CONFIG_TEST_DOTENV_NAME") }) // godotenv sets the process environment

	require.NoError(t, LoadConfig([]Value{String("CONFIG_TEST_DOTENV_NAME")}))

	require.Equal(t, "from-dotenv", Get().String("CONFIG_TEST_DOTENV_NAME"))
}

func TestTheEnvironmentWinsOverDotEnvFiles(t *testing.T) {
	directory := t.TempDir()
	writeFile(t, filepath.Join(directory, "app.env"), "CONFIG_TEST_DOTENV_NAME=from-dotenv\n")
	t.Setenv("CONFIG_TEST_DOTENV_NAME", "from-environment")

	require.NoError(t, LoadConfigWithOptions([]Value{String("CONFIG_TEST_DOTENV_NAME")}, &LoadConfigOptions{
		DotEnvFiles: []string{filepath.Join(directory, "app.env")},
	}))

	require.Equal(t, "from-environment", Get().String("CONFIG_TEST_DOTENV_NAME"))
}

func TestMissingDotEnvFilesAreSkippedAndTheFirstFileWins(t *testing.T) {
	directory := t.TempDir()
	writeFile(t, filepath.Join(directory, "first.env"), "CONFIG_TEST_DOTENV_NAME=first\n")
	writeFile(t, filepath.Join(directory, "second.env"), "CONFIG_TEST_DOTENV_NAME=second\nCONFIG_TEST_DOTENV_PORT=7070\n")
	t.Cleanup(func() {
		_ = os.Unsetenv("CONFIG_TEST_DOTENV_NAME") // godotenv sets the process environment
		_ = os.Unsetenv("CONFIG_TEST_DOTENV_PORT")
	})

	require.NoError(t, LoadConfigWithOptions([]Value{
		String("CONFIG_TEST_DOTENV_NAME"),
		Int("CONFIG_TEST_DOTENV_PORT"),
	}, &LoadConfigOptions{
		DotEnvFile:  filepath.Join(directory, "missing.env"),
		DotEnvFiles: []string{filepath.Join(directory, "first.env"), filepath.Join(directory, "second.env")},
	}))

	require.Equal(t, "first", Get().String("CONFIG_TEST_DOTENV_NAME"))
	require.Equal(t, 7070, Get().Int("CONFIG_TEST_DOTENV_PORT"))
}

func TestAnUnreadableDotEnvFileFails(t *testing.T) {
	directory := t.TempDir()

	err := LoadConfigWithOptions([]Value{}, &LoadConfigOptions{DotEnvFiles: []string{directory}})

	require.ErrorContains(t, err, "load dotenv file "+directory)
}

func TestDotEnvFilesDefaultToDotEnv(t *testing.T) {
	require.Equal(t, []string{".env"}, dotEnvFiles(nil))
	require.Equal(t, []string{".env"}, dotEnvFiles(&LoadConfigOptions{}))
	require.Equal(t, []string{"a.env", "b.env"}, dotEnvFiles(&LoadConfigOptions{DotEnvFile: "a.env", DotEnvFiles: []string{"b.env"}}))
}
