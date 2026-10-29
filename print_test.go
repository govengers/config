package config

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func printed(t *testing.T) string {
	t.Helper()
	var output bytes.Buffer
	require.NoError(t, Fprint(&output))
	return output.String()
}

func TestPrintMasksSensitiveValuesWithoutRevealingTheirLength(t *testing.T) {
	t.Setenv("CONFIG_TEST_PASSWORD", "correct horse battery staple")
	t.Setenv("CONFIG_TEST_PIN", "1234")
	require.NoError(t, LoadConfig([]Value{
		String("CONFIG_TEST_PASSWORD").Sensitive(),
		Int("CONFIG_TEST_PIN").Sensitive(),
		String("CONFIG_TEST_TOKEN").Default("default-secret").Sensitive(),
	}))

	output := printed(t)

	require.NotContains(t, output, "correct horse")
	require.NotContains(t, output, "1234")
	require.NotContains(t, output, "default-secret")
	require.NotContains(t, output, strings.Repeat("*", len("correct horse battery staple")))
	require.Contains(t, output, maskedValue)
}

func TestPrintShowsTypesDefaultsAndWhetherAValueWasProvided(t *testing.T) {
	t.Setenv("CONFIG_TEST_ORIGINS", "https://a.example,https://b.example")
	require.NoError(t, LoadConfig([]Value{
		StringArray("CONFIG_TEST_ORIGINS"),
		Int("CONFIG_TEST_PORT").Default(8080),
	}))

	output := printed(t)

	require.Contains(t, output, "Application Configuration")
	require.Contains(t, output, "string array")
	require.Contains(t, output, "https://a.example, https://b.example")
	require.Contains(t, output, "8080")
	require.Contains(t, output, "true")
	require.Contains(t, output, "false")
	require.Less(t, strings.Index(output, "CONFIG_TEST_ORIGINS"), strings.Index(output, "CONFIG_TEST_PORT"))
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }

func TestFprintReturnsTheWriteError(t *testing.T) {
	require.NoError(t, LoadConfig([]Value{Int("CONFIG_TEST_PORT").Default(8080)}))

	require.EqualError(t, Fprint(failingWriter{}), "disk full")
}

func TestFormatDropsControlCharacters(t *testing.T) {
	require.Equal(t, "a[31mbc", format("a\x1b[31mb\nc"))
	require.Equal(t, "x, y", format([]string{"x", "y\r"}))
}

func TestTruncate(t *testing.T) {
	short := strings.Repeat("a", maximumDisplayLength)
	long := strings.Repeat("ä", maximumDisplayLength+1)

	require.Equal(t, short, truncate(short))
	require.Equal(t, strings.Repeat("ä", maximumDisplayLength-3)+"...", truncate(long))
}
