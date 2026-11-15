package config

import (
	"math"
	"net/url"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

type typedConfiguration struct {
	Timeout  time.Duration
	Ratio    float64
	BaseURL  url.URL
	LogLevel string
}

func typedValues(configuration *typedConfiguration) []Value {
	return []Value{
		Duration("CONFIG_TEST_TIMEOUT").Default(30 * time.Second).Into(&configuration.Timeout),
		Float("CONFIG_TEST_RATIO").Default(0.5).Into(&configuration.Ratio),
		URL("CONFIG_TEST_BASE_URL").Schemes("https", "HTTP").Default("http://localhost:8080").Into(&configuration.BaseURL),
		Enum("CONFIG_TEST_LOG_LEVEL", "debug", "info", "warn").Default("info").Into(&configuration.LogLevel),
	}
}

func TestIntoBindsTheNewTypesFromTheEnvironment(t *testing.T) {
	t.Setenv("CONFIG_TEST_TIMEOUT", "1h30m")
	t.Setenv("CONFIG_TEST_RATIO", "-2.5e-3")
	t.Setenv("CONFIG_TEST_BASE_URL", "HTTPS://user:pass@api.example:8443/v1?x=1")
	t.Setenv("CONFIG_TEST_LOG_LEVEL", "debug")
	var configuration typedConfiguration

	require.NoError(t, LoadConfig(typedValues(&configuration)))

	require.Equal(t, 90*time.Minute, configuration.Timeout)
	require.InDelta(t, -0.0025, configuration.Ratio, 1e-12)
	require.Equal(t, "https://user:pass@api.example:8443/v1?x=1", configuration.BaseURL.String())
	require.Equal(t, "debug", configuration.LogLevel)
	require.Equal(t, 90*time.Minute, Get().Duration("CONFIG_TEST_TIMEOUT"))
	require.InDelta(t, -0.0025, Get().Float("CONFIG_TEST_RATIO"), 1e-12)
	readURL := Get().URL("CONFIG_TEST_BASE_URL")
	require.Equal(t, "api.example:8443", readURL.Host)
	require.Equal(t, "debug", Get().Enum("CONFIG_TEST_LOG_LEVEL"))
}

func TestTheNewTypesUseTheirDefaults(t *testing.T) {
	var configuration typedConfiguration

	require.NoError(t, LoadConfig(typedValues(&configuration)))

	require.Equal(t, typedConfiguration{
		Timeout:  30 * time.Second,
		Ratio:    0.5,
		BaseURL:  url.URL{Scheme: "http", Host: "localhost:8080"},
		LogLevel: "info",
	}, configuration)
}

func TestInvalidValuesOfTheNewTypesAreAllReportedAtOnce(t *testing.T) {
	t.Setenv("CONFIG_TEST_TIMEOUT", "300")
	t.Setenv("CONFIG_TEST_RATIO", "NaN")
	t.Setenv("CONFIG_TEST_BASE_URL", "ftp://files.example")
	t.Setenv("CONFIG_TEST_LOG_LEVEL", "INFO")
	var configuration typedConfiguration

	err := LoadConfig(typedValues(&configuration))

	require.ErrorIs(t, err, ErrInvalid)
	require.ErrorContains(t, err, `CONFIG_TEST_TIMEOUT: "300" is not a duration such as 30s, 5m or 1h30m`)
	require.ErrorContains(t, err, `CONFIG_TEST_RATIO: "NaN" is not a finite number`)
	require.ErrorContains(t, err, `CONFIG_TEST_BASE_URL: "ftp://files.example" has a scheme other than https, http`)
	require.ErrorContains(t, err, `CONFIG_TEST_LOG_LEVEL: "INFO" is not one of debug, info, warn`)
	require.Equal(t, typedConfiguration{}, configuration)
}

func TestParseURL(t *testing.T) {
	testCases := map[string]struct {
		raw       string
		wantHost  string
		wantError error
	}{
		"https":                {raw: "https://api.example/v1", wantHost: "api.example"},
		"postgres":             {raw: "postgres://user:secret@db:5432/app?sslmode=disable", wantHost: "db:5432"},
		"empty":                {raw: ""},
		"host without scheme":  {raw: "api.example/v1", wantError: ErrInvalid},
		"host and port only":   {raw: "localhost:8080", wantError: ErrInvalid},
		"scheme without host":  {raw: "file:///etc/app", wantError: ErrInvalid},
		"unparsable":           {raw: "http://[::1", wantError: ErrInvalid},
		"whitespace only":      {raw: "  ", wantError: ErrInvalid},
		"relative path":        {raw: "/v1/items", wantError: ErrInvalid},
		"credentials in error": {raw: "http://user:p%zz@host", wantError: ErrInvalid},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			got, err := parseURL(testCase.raw)

			require.ErrorIs(t, err, testCase.wantError)
			require.Equal(t, testCase.wantHost, got.Host)
			if err != nil {
				require.NotContains(t, err.Error(), testCase.raw, "url.Parse quotes the value")
			}
		})
	}
}

func TestParseFloatAndDuration(t *testing.T) {
	ratio, err := parseFloat("1e3")
	require.NoError(t, err)
	require.InDelta(t, 1000.0, ratio, 0)
	_, err = parseFloat("one")
	require.ErrorIs(t, err, ErrInvalid)

	zero, err := parseDuration("0")
	require.NoError(t, err)
	require.Zero(t, zero)
	negative, err := parseDuration("-1.5s")
	require.NoError(t, err)
	require.Equal(t, -1500*time.Millisecond, negative)
	_, err = parseDuration("")
	require.ErrorIs(t, err, ErrInvalid)
}

func TestInfiniteFloatsAreRejected(t *testing.T) {
	t.Setenv("CONFIG_TEST_RATIO", "-Inf")

	err := LoadConfig([]Value{Float("CONFIG_TEST_RATIO")})

	require.ErrorIs(t, err, ErrInvalid)
	require.ErrorContains(t, err, "is not a finite number")
}

func TestAnEmptyURLIsTheZeroURLUnlessNotEmpty(t *testing.T) {
	t.Setenv("CONFIG_TEST_OPTIONAL_URL", "")
	t.Setenv("CONFIG_TEST_REQUIRED_URL", "")
	optional := url.URL{Scheme: "https", Host: "stale.example"}

	require.NoError(t, LoadConfig([]Value{URL("CONFIG_TEST_OPTIONAL_URL").Schemes("https").Into(&optional)}))
	require.Equal(t, url.URL{}, optional)

	err := LoadConfig([]Value{URL("CONFIG_TEST_REQUIRED_URL").NotEmpty()})
	require.ErrorIs(t, err, ErrInvalid)
	require.ErrorContains(t, err, "CONFIG_TEST_REQUIRED_URL must not be empty")
}

func TestInvalidDefaultsFailEvenWhenTheVariableIsSet(t *testing.T) {
	t.Setenv("CONFIG_TEST_BASE_URL", "https://api.example")
	t.Setenv("CONFIG_TEST_LOG_LEVEL", "info")

	err := LoadConfig([]Value{
		URL("CONFIG_TEST_BASE_URL").Default("localhost:8080"),
		URL("CONFIG_TEST_CALLBACK_URL").Schemes("https").Default("http://insecure.example"),
		URL("CONFIG_TEST_EMPTY_URL").NotEmpty().Default(""),
		Enum("CONFIG_TEST_LOG_LEVEL", "debug", "info").Default("verbose"),
		Float("CONFIG_TEST_RATIO").Default(math.Inf(1)),
	})

	require.ErrorIs(t, err, ErrInvalid)
	require.ErrorContains(t, err, `CONFIG_TEST_BASE_URL has an invalid default: "localhost:8080" is not an absolute URL`)
	require.ErrorContains(t, err, `CONFIG_TEST_CALLBACK_URL has an invalid default: "http://insecure.example" has a scheme other than https`)
	require.ErrorContains(t, err, "CONFIG_TEST_EMPTY_URL is NotEmpty but its default is empty")
	require.ErrorContains(t, err, `CONFIG_TEST_LOG_LEVEL has an invalid default: "verbose" is not one of debug, info`)
	require.ErrorContains(t, err, "CONFIG_TEST_RATIO has an invalid default")
}

func TestAnEnumWithoutAllowedValuesFails(t *testing.T) {
	err := LoadConfig([]Value{Enum("CONFIG_TEST_LOG_LEVEL").Default("")})

	require.ErrorIs(t, err, ErrInvalid)
	require.ErrorContains(t, err, "CONFIG_TEST_LOG_LEVEL is an Enum without allowed values")
}

func TestSensitiveValuesOfTheNewTypesStayOutOfErrors(t *testing.T) {
	t.Setenv("CONFIG_TEST_SECRET_URL", "ftp://admin:hunter2@files.example")
	t.Setenv("CONFIG_TEST_SECRET_MODE", "hunter3")

	err := LoadConfig([]Value{
		URL("CONFIG_TEST_SECRET_URL").Schemes("https").Sensitive(),
		Enum("CONFIG_TEST_SECRET_MODE", "a", "b").Sensitive(),
		URL("CONFIG_TEST_SECRET_DEFAULT").Default("admin:hunter4@nohost").Sensitive(),
	})

	require.ErrorIs(t, err, ErrInvalid)
	for _, secret := range []string{"hunter2", "hunter3", "hunter4"} {
		require.NotContains(t, err.Error(), secret)
	}
	require.ErrorContains(t, err, "CONFIG_TEST_SECRET_URL: the value (sensitive, not shown) has a scheme other than https")
}

func TestIntoANilPointerOfTheNewTypesFails(t *testing.T) {
	var nilDuration *time.Duration
	var nilFloat *float64
	var nilURL *url.URL

	err := LoadConfig([]Value{
		Duration("CONFIG_TEST_TIMEOUT").Default(time.Second).Into(nilDuration),
		Float("CONFIG_TEST_RATIO").Default(1).Into(nilFloat),
		URL("CONFIG_TEST_BASE_URL").Default("").Into(nilURL),
	})

	require.ErrorIs(t, err, ErrInvalid)
	for _, name := range []string{"CONFIG_TEST_TIMEOUT", "CONFIG_TEST_RATIO", "CONFIG_TEST_BASE_URL"} {
		require.ErrorContains(t, err, name+" is bound with Into to a nil pointer")
	}
}

func TestSetChangesTheNewTypesAndAppliesTheirRules(t *testing.T) {
	var configuration typedConfiguration
	require.NoError(t, LoadConfig(typedValues(&configuration)))

	Set().Duration("CONFIG_TEST_TIMEOUT", time.Minute)
	Set().Float("CONFIG_TEST_RATIO", 0.25)
	Set().URL("CONFIG_TEST_BASE_URL", "https://set.example")
	Set().Enum("CONFIG_TEST_LOG_LEVEL", "warn")

	require.Equal(t, typedConfiguration{
		Timeout:  time.Minute,
		Ratio:    0.25,
		BaseURL:  url.URL{Scheme: "https", Host: "set.example"},
		LogLevel: "warn",
	}, configuration)
	require.PanicsWithError(t, "config: CONFIG_TEST_LOG_LEVEL is not one of debug, info, warn: invalid", func() { Set().Enum("CONFIG_TEST_LOG_LEVEL", "trace") })
	require.PanicsWithError(t, "config: CONFIG_TEST_BASE_URL has a scheme other than https, http: invalid", func() { Set().URL("CONFIG_TEST_BASE_URL", "ws://set.example") })
	require.PanicsWithError(t, "config: CONFIG_TEST_BASE_URL is not an absolute URL with scheme and host: invalid", func() { Set().URL("CONFIG_TEST_BASE_URL", "set.example") })
	require.Panics(t, func() { Set().Float("CONFIG_TEST_RATIO", math.NaN()) })
	require.Panics(t, func() { Set().String("CONFIG_TEST_LOG_LEVEL", "info") }, "an Enum is not a String")
	require.Equal(t, "warn", configuration.LogLevel)
}

func TestGettersOfTheNewTypesPanicOnTheWrongType(t *testing.T) {
	require.NoError(t, LoadConfig([]Value{String("CONFIG_TEST_NAME").Default("x")}))

	require.Panics(t, func() { Get().Duration("CONFIG_TEST_NAME") })
	require.Panics(t, func() { Get().Float("CONFIG_TEST_NAME") })
	require.Panics(t, func() { Get().URL("CONFIG_TEST_NAME") })
	require.PanicsWithError(t, "config: CONFIG_TEST_NAME is declared as string, not as enum", func() { Get().Enum("CONFIG_TEST_NAME") })
}

func TestBindPFlagParsesAndChecksTheNewTypes(t *testing.T) {
	var configuration typedConfiguration
	require.NoError(t, LoadConfig(typedValues(&configuration)))
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.Duration("timeout", 0, "")
	flags.Float64("ratio", 0, "")
	flags.String("base-url", "", "")
	flags.String("log-level", "", "")
	require.NoError(t, flags.Parse([]string{"--timeout=2m", "--ratio=0.75", "--base-url=https://flag.example", "--log-level=trace"}))

	require.NoError(t, BindPFlag("CONFIG_TEST_TIMEOUT", flags.Lookup("timeout")))
	require.NoError(t, BindPFlag("CONFIG_TEST_RATIO", flags.Lookup("ratio")))
	require.NoError(t, BindPFlag("CONFIG_TEST_BASE_URL", flags.Lookup("base-url")))
	err := BindPFlag("CONFIG_TEST_LOG_LEVEL", flags.Lookup("log-level"))

	require.ErrorIs(t, err, ErrInvalid)
	require.Equal(t, typedConfiguration{
		Timeout:  2 * time.Minute,
		Ratio:    0.75,
		BaseURL:  url.URL{Scheme: "https", Host: "flag.example"},
		LogLevel: "info",
	}, configuration)
}

func TestPrintShowsTheNewTypesAndRedactsURLPasswords(t *testing.T) {
	t.Setenv("CONFIG_TEST_BASE_URL", "https://user:url-password@api.example/v1")
	var configuration typedConfiguration
	require.NoError(t, LoadConfig(typedValues(&configuration)))

	output := printed(t)

	require.NotContains(t, output, "url-password")
	require.Contains(t, output, "https://user:xxxxx@api.example/v1")
	require.Contains(t, output, "http://localhost:8080")
	require.Contains(t, output, "30s")
	require.Contains(t, output, "0.5")
	for _, typeName := range []string{"duration", "float", "url", "enum"} {
		require.Contains(t, output, typeName)
	}
}

func TestPrintMasksSensitiveValuesOfTheNewTypes(t *testing.T) {
	require.NoError(t, LoadConfig([]Value{
		Duration("CONFIG_TEST_SECRET_TIMEOUT").Default(4711 * time.Second).Sensitive(),
		Float("CONFIG_TEST_SECRET_RATIO").Default(47.12).Sensitive(),
		URL("CONFIG_TEST_SECRET_URL").Default("https://user@secret-host.example").Sensitive(),
	}))

	output := printed(t)

	for _, secret := range []string{"4711", "47.12", "secret-host"} {
		require.NotContains(t, output, secret)
	}
}
