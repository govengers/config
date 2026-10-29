package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadReadsEveryTypeFromTheEnvironment(t *testing.T) {
	t.Setenv("CONFIG_TEST_NAME", "service")
	t.Setenv("CONFIG_TEST_ORIGINS", "https://a.example, https://b.example")
	t.Setenv("CONFIG_TEST_DEBUG", "TRUE")
	t.Setenv("CONFIG_TEST_PORT", "9090")

	require.NoError(t, LoadConfig([]Value{
		String("CONFIG_TEST_NAME"),
		StringArray("CONFIG_TEST_ORIGINS"),
		Bool("CONFIG_TEST_DEBUG"),
		Int("CONFIG_TEST_PORT"),
	}))

	require.Equal(t, "service", Get().String("CONFIG_TEST_NAME"))
	require.Equal(t, []string{"https://a.example", "https://b.example"}, Get().StringArray("CONFIG_TEST_ORIGINS"))
	require.True(t, Get().Bool("CONFIG_TEST_DEBUG"))
	require.Equal(t, 9090, Get().Int("CONFIG_TEST_PORT"))
}

func TestDefaultsApplyWhenTheVariableIsUnset(t *testing.T) {
	require.NoError(t, LoadConfig([]Value{
		String("CONFIG_TEST_UNSET_NAME").Default("fallback"),
		StringArray("CONFIG_TEST_UNSET_ORIGINS").Default([]string{"https://default.example"}),
		Bool("CONFIG_TEST_UNSET_DEBUG").Default(true),
		Int("CONFIG_TEST_UNSET_PORT").Default(8080),
	}))

	require.Equal(t, "fallback", Get().String("CONFIG_TEST_UNSET_NAME"))
	require.Equal(t, []string{"https://default.example"}, Get().StringArray("CONFIG_TEST_UNSET_ORIGINS"))
	require.True(t, Get().Bool("CONFIG_TEST_UNSET_DEBUG"))
	require.Equal(t, 8080, Get().Int("CONFIG_TEST_UNSET_PORT"))
}

func TestAnEmptyVariableIsProvidedAndWinsOverTheDefault(t *testing.T) {
	t.Setenv("CONFIG_TEST_EMPTY", "")

	require.NoError(t, LoadConfig([]Value{String("CONFIG_TEST_EMPTY").Default("fallback")}))

	require.Empty(t, Get().String("CONFIG_TEST_EMPTY"))
}

func TestAMissingRequiredValueFails(t *testing.T) {
	err := LoadConfig([]Value{String("CONFIG_TEST_MISSING")})

	require.ErrorIs(t, err, ErrMissing)
	require.ErrorContains(t, err, "CONFIG_TEST_MISSING")
}

func TestLoadReportsEveryProblemAtOnce(t *testing.T) {
	t.Setenv("CONFIG_TEST_BAD_PORT", "eighty")
	t.Setenv("CONFIG_TEST_BAD_DEBUG", "yes")

	err := LoadConfig([]Value{
		String("CONFIG_TEST_MISSING_A"),
		String("CONFIG_TEST_MISSING_B"),
		Int("CONFIG_TEST_BAD_PORT"),
		Bool("CONFIG_TEST_BAD_DEBUG"),
	})

	require.ErrorIs(t, err, ErrMissing)
	require.ErrorIs(t, err, ErrInvalid)
	for _, name := range []string{"CONFIG_TEST_MISSING_A", "CONFIG_TEST_MISSING_B", "CONFIG_TEST_BAD_PORT", "CONFIG_TEST_BAD_DEBUG"} {
		require.ErrorContains(t, err, name)
	}
}

func TestNotEmptyRejectsAnEmptyValue(t *testing.T) {
	t.Setenv("CONFIG_TEST_EMPTY_STRING", "")
	t.Setenv("CONFIG_TEST_EMPTY_LIST", " , ,")

	err := LoadConfig([]Value{
		String("CONFIG_TEST_EMPTY_STRING").NotEmpty(),
		StringArray("CONFIG_TEST_EMPTY_LIST").NotEmpty(),
	})

	require.ErrorIs(t, err, ErrInvalid)
	require.ErrorContains(t, err, "CONFIG_TEST_EMPTY_STRING must not be empty")
	require.ErrorContains(t, err, "CONFIG_TEST_EMPTY_LIST must not be empty")
}

func TestNotEmptyRejectsAnEmptyDefault(t *testing.T) {
	err := LoadConfig([]Value{
		String("CONFIG_TEST_UNSET_STRING").NotEmpty().Default(""),
		StringArray("CONFIG_TEST_UNSET_LIST").NotEmpty().Default(nil),
	})

	require.ErrorIs(t, err, ErrInvalid)
	require.ErrorContains(t, err, "CONFIG_TEST_UNSET_STRING is NotEmpty but its default is empty")
	require.ErrorContains(t, err, "CONFIG_TEST_UNSET_LIST is NotEmpty but its default is empty")
}

func TestADuplicateDeclarationFails(t *testing.T) {
	t.Setenv("CONFIG_TEST_TWICE", "value")

	err := LoadConfig([]Value{String("CONFIG_TEST_TWICE"), Int("CONFIG_TEST_TWICE")})

	require.ErrorIs(t, err, ErrInvalid)
	require.ErrorContains(t, err, "CONFIG_TEST_TWICE is declared twice")
}

func TestParseBool(t *testing.T) {
	testCases := []struct {
		raw       string
		want      bool
		wantError error
	}{
		{raw: "true", want: true},
		{raw: "TRUE", want: true},
		{raw: "False", want: false},
		{raw: "yes", wantError: ErrInvalid},
		{raw: "1", wantError: ErrInvalid},
		{raw: "", wantError: ErrInvalid},
	}
	for _, testCase := range testCases {
		t.Run(testCase.raw, func(t *testing.T) {
			got, err := parseBool(testCase.raw)
			require.ErrorIs(t, err, testCase.wantError)
			require.Equal(t, testCase.want, got)
		})
	}
}

func TestParseInt(t *testing.T) {
	testCases := []struct {
		raw       string
		want      int
		wantError error
	}{
		{raw: "8080", want: 8080},
		{raw: "-1", want: -1},
		{raw: "eighty", wantError: ErrInvalid},
		{raw: "1.5", wantError: ErrInvalid},
		{raw: "", wantError: ErrInvalid},
	}
	for _, testCase := range testCases {
		t.Run(testCase.raw, func(t *testing.T) {
			got, err := parseInt(testCase.raw)
			require.ErrorIs(t, err, testCase.wantError)
			require.Equal(t, testCase.want, got)
		})
	}
}

func TestParseStringArray(t *testing.T) {
	testCases := []struct {
		name string
		raw  string
		want []string
	}{
		{name: "splits at commas", raw: "a,b,c", want: []string{"a", "b", "c"}},
		{name: "trims items", raw: " a , b ", want: []string{"a", "b"}},
		{name: "drops empty and blank items", raw: "a,, ,b,", want: []string{"a", "b"}},
		{name: "empty input is an empty list", raw: "", want: []string{}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.want, parseStringArray(testCase.raw))
		})
	}
}

func TestParseRejectsAnUnknownType(t *testing.T) {
	_, err := parse(valueType("duration"), "5s")

	require.ErrorIs(t, err, ErrInvalid)
}
