// Command app is the program the end-to-end tests build and run: it loads its configuration the way
// a service does in main, prints the table to stderr and the bound struct as JSON to stdout.
package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/govengers/config/v2"
	"github.com/spf13/pflag"
)

// Configuration holds every value the app reads.
type Configuration struct {
	ServiceName    string
	Port           int
	DevMode        bool
	DatabaseURL    string
	AllowedOrigins []string
	Timeout        time.Duration
	SampleRate     float64
	PublicURL      url.URL
	LogLevel       string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	flags := pflag.NewFlagSet("app", pflag.ContinueOnError)
	flags.Int("port", 0, "overrides PORT")
	dotEnvFiles := flags.StringSlice("dotenv-file", nil, "dotenv files to read")
	readUndeclared := flags.Bool("read-undeclared", false, "read a key that was never declared")
	if err := flags.Parse(arguments); err != nil {
		return err
	}

	configuration := &Configuration{}
	err := config.LoadConfigWithOptions([]config.Value{
		config.String("SERVICE_NAME").Default("e2e-app").Into(&configuration.ServiceName),
		config.Int("PORT").Default(8080).Into(&configuration.Port),
		config.Bool("DEV_MODE").Default(false).Into(&configuration.DevMode),
		config.String("DATABASE_URL").NotEmpty().Sensitive().Into(&configuration.DatabaseURL),
		config.StringArray("ALLOWED_ORIGINS").Default([]string{"http://localhost"}).Into(&configuration.AllowedOrigins),
		config.Duration("TIMEOUT").Default(10 * time.Second).Into(&configuration.Timeout),
		config.Float("SAMPLE_RATE").Default(1).Into(&configuration.SampleRate),
		config.URL("PUBLIC_URL").Schemes("https", "http").Default("http://localhost:8080").Into(&configuration.PublicURL),
		config.Enum("LOG_LEVEL", "debug", "info", "warn", "error").Default("info").Into(&configuration.LogLevel),
	}, &config.LoadConfigOptions{DotEnvFiles: *dotEnvFiles})
	if err != nil {
		return err
	}
	if err := config.BindPFlag("PORT", flags.Lookup("port")); err != nil {
		return err
	}
	if *readUndeclared {
		_ = config.Get().String("NOT_DECLARED") // panics: the key was never declared
	}

	if err := config.Fprint(os.Stderr); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(configuration)
}
