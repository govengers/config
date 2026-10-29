package config

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

func parsedFlags(t *testing.T, arguments ...string) *pflag.FlagSet {
	t.Helper()
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("name", "flag-default", "")
	flags.StringSlice("origins", nil, "")
	flags.Bool("debug", false, "")
	flags.Int("port", 0, "")
	flags.String("port-text", "", "")
	require.NoError(t, flags.Parse(arguments))
	return flags
}

func TestBindPFlagOverridesEveryTypeWhenTheFlagIsSet(t *testing.T) {
	var configuration testConfiguration
	require.NoError(t, LoadConfig(testValues(&configuration)))
	flags := parsedFlags(t, "--name=from-flag", "--origins=https://x.example,https://y.example", "--debug", "--port=9999")

	require.NoError(t, BindPFlag("CONFIG_TEST_NAME", flags.Lookup("name")))
	require.NoError(t, BindPFlag("CONFIG_TEST_ORIGINS", flags.Lookup("origins")))
	require.NoError(t, BindPFlag("CONFIG_TEST_DEBUG", flags.Lookup("debug")))
	require.NoError(t, BindPFlag("CONFIG_TEST_PORT", flags.Lookup("port")))

	require.Equal(t, testConfiguration{
		Name:    "from-flag",
		Origins: []string{"https://x.example", "https://y.example"},
		Debug:   true,
		Port:    9999,
	}, configuration)
	require.Equal(t, 9999, Get().Int("CONFIG_TEST_PORT"))
}

func TestBindPFlagKeepsTheEnvironmentValueWhenTheFlagIsNotSet(t *testing.T) {
	t.Setenv("CONFIG_TEST_NAME", "from-environment")
	var configuration testConfiguration
	require.NoError(t, LoadConfig(testValues(&configuration)))

	require.NoError(t, BindPFlag("CONFIG_TEST_NAME", parsedFlags(t).Lookup("name")))

	require.Equal(t, "from-environment", configuration.Name)
}

func TestBindPFlagEnforcesNotEmpty(t *testing.T) {
	name := ""
	require.NoError(t, LoadConfig([]Value{String("CONFIG_TEST_NAME").NotEmpty().Default("from-default").Into(&name)}))

	err := BindPFlag("CONFIG_TEST_NAME", parsedFlags(t, "--name=").Lookup("name"))

	require.ErrorIs(t, err, ErrInvalid)
	require.Equal(t, "from-default", name)
	require.Equal(t, "from-default", Get().String("CONFIG_TEST_NAME"))
}

func TestBindPFlagHidesASensitiveValue(t *testing.T) {
	require.NoError(t, LoadConfig([]Value{Int("CONFIG_TEST_PIN").Default(0).Sensitive()}))
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("pin-text", "", "")
	require.NoError(t, flags.Parse([]string{"--pin-text=4711x"}))

	err := BindPFlag("CONFIG_TEST_PIN", flags.Lookup("pin-text"))

	require.ErrorIs(t, err, ErrInvalid)
	require.NotContains(t, err.Error(), "4711")
}

func TestBindPFlagParsesListFlagsLikeTheEnvironment(t *testing.T) {
	var configuration testConfiguration
	require.NoError(t, LoadConfig(testValues(&configuration)))

	require.NoError(t, BindPFlag("CONFIG_TEST_ORIGINS", parsedFlags(t, "--origins= https://a.example , ,https://b.example").Lookup("origins")))

	require.Equal(t, []string{"https://a.example", "https://b.example"}, configuration.Origins)
}

func TestBindPFlagIgnoresANilFlag(t *testing.T) {
	require.NoError(t, BindPFlag("CONFIG_TEST_ANYTHING", nil))
}

func TestBindPFlagRejectsAnInvalidValue(t *testing.T) {
	var configuration testConfiguration
	require.NoError(t, LoadConfig(testValues(&configuration)))

	err := BindPFlag("CONFIG_TEST_PORT", parsedFlags(t, "--port-text=eighty").Lookup("port-text"))

	require.ErrorIs(t, err, ErrInvalid)
	require.ErrorContains(t, err, "flag --port-text for CONFIG_TEST_PORT")
	require.Equal(t, 8080, configuration.Port)
}

func TestBindPFlagRejectsAnUndeclaredKey(t *testing.T) {
	require.NoError(t, LoadConfig([]Value{}))

	err := BindPFlag("CONFIG_TEST_TYPO", parsedFlags(t, "--name=x").Lookup("name"))

	require.EqualError(t, err, "config: CONFIG_TEST_TYPO was never declared")
}
