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

import (
	"log"

	"github.com/govengers/config/v2"
)

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
	log.Printf("listening on :%d", configuration.Port) // copy fields into the options of your components
}
```

| Builder | Environment value | Methods |
|---------|-------------------|---------|
| `config.String(name)` | any text | `Default`, `NotEmpty`, `Sensitive`, `Into(*string)` |
| `config.StringArray(name)` | comma-separated; items are trimmed, empty items dropped | `Default`, `NotEmpty`, `Sensitive`, `Into(*[]string)` |
| `config.Bool(name)` | `true` or `false`, any case | `Default`, `Sensitive`, `Into(*bool)` |
| `config.Int(name)` | an integer | `Default`, `Sensitive`, `Into(*int)` |

- A value without `Default` is required.
- `NotEmpty` rejects an empty or whitespace-only value and an empty default.
- `Sensitive` masks the value in `Print` with a fixed `****`, so not even its length shows, and keeps it out of every error `LoadConfig` and `BindPFlag` return.

`config.Print()` writes the table to stdout (in color), `config.Fprint(writer)` anywhere else and returns the write error. Control characters in values are dropped:

```
 Application Configuration
 ENV VAR          TYPE          DEFAULT VALUE          VALUE PROVIDED  CURRENT VALUE
 ALLOWED_ORIGINS  string array  http://localhost:9000  false           http://localhost:9000
 DATABASE_URL     string        -                      true            ****
 DEV_MODE         bool          false                  false           false
 LOG_LEVEL        string        info                   false           info
 PORT             int           8080                   true            9000
```

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

A variable already set in the environment wins over every file, and an earlier file wins over a later one. A file that exists but cannot be read or parsed is an error. The files are validated together with everything else: only after a successful load does `LoadConfig` set their variables, declared or not, in the process environment, for other code that reads it.

- Only regular files of at most 1 MiB are read; symlinks are followed, so a dotenv file mounted from a Kubernetes Secret works.
- A syntax error names the file but not its content, because the content holds secrets.
- In unquoted and double-quoted values, `$NAME` and `${NAME}` expand to earlier keys of the same file; quote secrets with single quotes.
- `DisableDotEnv: true` reads no file at all. Use it in production when the configuration comes from the environment only: a `.env` in the working directory can set any variable of the process, so that directory must not be writable by anyone but the operator.

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
| invalid values were quoted in errors, secrets included | a `Sensitive` value never appears in an error; dotenv syntax errors no longer quote the file |
| `LoadConfig` stopped at the first problem and left partial state | reports every problem at once; on failure nothing changes, the process environment included |
| `BindPFlag` applied a flag's default even when the flag was not set, and panicked | applies only set flags, supports `StringArray`, enforces `NotEmpty`, returns an error |
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
