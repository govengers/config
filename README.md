# config

[![CI](https://github.com/govengers/config/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/govengers/config/actions/workflows/ci.yml)
[![unit tests](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/govengers/config/badges/unit-tests.json)](https://github.com/govengers/config/actions/workflows/ci.yml)
[![e2e tests](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/govengers/config/badges/e2e-tests.json)](test/e2e)
[![coverage](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/govengers/config/badges/coverage.json)](https://github.com/govengers/config/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/govengers/config/v2.svg)](https://pkg.go.dev/github.com/govengers/config/v2)
[![License](https://img.shields.io/github/license/govengers/config)](LICENSE)

Environment-based configuration for Go services: declared once, validated at startup, bound into your own struct.

A service in a crash loop at startup beats one that crashes the first time a request needs a missing value. `LoadConfig` fails with every missing or invalid value listed, and reading a key that was never declared panics. Load and read everything in `main`, and both happen before the service does anything else.

## Installation

```bash
go get github.com/govengers/config/v2
```

## Usage

Declare every value once and bind it into a struct with `Into`:

```go
package main

import "github.com/govengers/config/v2"

type Configuration struct {
	LogLevel       string
	DevMode        bool
	Port           int
	DatabaseURL    string
	AllowedOrigins []string
}

func loadConfiguration() (*Configuration, error) {
	configuration := &Configuration{}
	err := config.LoadConfig([]config.Value{
		config.String("LOG_LEVEL").NotEmpty().Default("info").Into(&configuration.LogLevel),
		config.Bool("DEV_MODE").Default(false).Into(&configuration.DevMode),
		config.Int("PORT").Default(8080).Into(&configuration.Port),
		config.String("DATABASE_URL").NotEmpty().Sensitive().Into(&configuration.DatabaseURL),
		config.StringArray("ALLOWED_ORIGINS").Default([]string{"http://localhost:9000"}).Into(&configuration.AllowedOrigins),
	})
	if err != nil {
		return nil, err
	}
	return configuration, nil
}

func main() {
	configuration, err := loadConfiguration()
	if err != nil {
		panic(err) // every missing or invalid value, in one error
	}
	config.Print()
	// copy configuration fields into the options of your components
}
```

| Builder | Environment value | Methods |
|---------|-------------------|---------|
| `config.String(name)` | any text | `Default`, `NotEmpty`, `Sensitive`, `Into(*string)` |
| `config.StringArray(name)` | comma-separated; items are trimmed, empty items dropped | `Default`, `NotEmpty`, `Sensitive`, `Into(*[]string)` |
| `config.Bool(name)` | `true` or `false`, any case | `Default`, `Sensitive`, `Into(*bool)` |
| `config.Int(name)` | an integer | `Default`, `Sensitive`, `Into(*int)` |

- A value without `Default` is required.
- `NotEmpty` rejects an empty value and an empty default.
- `Sensitive` masks the value in `Print` with a fixed `****`, so not even its length shows.

`config.Print()` writes the table to stdout, `config.Fprint(writer)` anywhere else:

![config.Print()](./images/configuration_printout.png)

### Reading by key

The registry behind `config.Get()` stays available. Its getters panic on a key that was never declared or that was declared with another type:

```go
port := config.Get().Int("PORT")
```

`config.Set()` changes a loaded value and its `Into` field, for tests.

### Dotenv files

`LoadConfig` reads `.env` from the working directory if it exists. `LoadConfigWithOptions` reads the files you list, in order, and skips missing ones:

```go
err := config.LoadConfigWithOptions(values, &config.LoadConfigOptions{
	DotEnvFiles: []string{"local.env", "shared.env"},
})
```

A variable already set in the environment wins over every file, and an earlier file wins over a later one. A file that exists but cannot be read or parsed is an error.

### Command line flags

`BindPFlag` overrides a loaded value with a [pflag](https://github.com/spf13/pflag) flag, but only if the flag was set:

```go
flags.Int("port", 0, "overrides PORT")
_ = flags.Parse(os.Args[1:])
if err := config.BindPFlag("PORT", flags.Lookup("port")); err != nil {
	panic(err)
}
```

## Tests

- Unit tests: `just test` (race detector, coverage gate at 80%).
- End-to-end tests: `just test-e2e` builds [`test/e2e/testdata/app`](test/e2e/testdata/app/main.go) and runs it as a process with real environment variables, dotenv files and flags.
- Everything CI runs: `just check`. CI uses the shared [govengers Go library workflow](https://github.com/govengers/.github).

## Migrating from v1 (`github.com/mxcd/go-config`)

1. Change the import from `github.com/mxcd/go-config/config` to `github.com/govengers/config/v2`. The package name stays `config`.
2. Check the breaking changes:

| v1 | v2 |
|----|----|
| `NotEmpty().Default("")` silently allowed an empty value | an error at load: drop `NotEmpty` if the value is optional |
| `LoadConfig` stopped at the first problem and left partial state | reports every problem at once; on failure nothing changes |
| `BindPFlag` applied a flag's default even when the flag was not set, and panicked | applies only set flags, supports `StringArray`, returns an error |
| a missing first dotenv file stopped the later ones from loading | missing files are skipped, every existing file is read |
| `" a, ,b"` parsed to `["a", "", "b"]` | parses to `["a", "b"]` |
| the library logged `.env` status with the standard logger | silent |
| `Print` labeled lists as `string` and masked secrets with one `*` per character | `string array`, fixed `****` |
| exported helpers `Descriptor`, `MaskSensitiveString`, `GetSanatizedDefaultValue`, `GetTruncated` | removed |
| `Get()` returned an unexported type | `*config.Registry`, safe for concurrent use |

3. Optionally bind into a struct with `Into` and read every value at startup, as shown above.

The v1 line stays on the [`v1`](https://github.com/govengers/config/tree/v1) branch.

## License

[Apache 2.0](LICENSE)
