package config_test

import (
	"fmt"
	"net/url"
	"time"

	"github.com/govengers/config/v2"
)

func ExampleLoadConfig() {
	type Configuration struct {
		Port           int
		DevMode        bool
		AllowedOrigins []string
	}
	configuration := &Configuration{}

	err := config.LoadConfig([]config.Value{
		config.Int("EXAMPLE_PORT").Default(8080).Into(&configuration.Port),
		config.Bool("EXAMPLE_DEV_MODE").Default(false).Into(&configuration.DevMode),
		config.StringArray("EXAMPLE_ALLOWED_ORIGINS").Default([]string{"https://app.example"}).Into(&configuration.AllowedOrigins),
	})
	if err != nil {
		panic(err) // a service crashes at startup, not later
	}

	fmt.Println(configuration.Port, configuration.DevMode, configuration.AllowedOrigins)
	// Output: 8080 false [https://app.example]
}

func ExampleGet() {
	err := config.LoadConfig([]config.Value{config.String("EXAMPLE_LOG_LEVEL").Default("info")})
	if err != nil {
		panic(err)
	}

	fmt.Println(config.Get().String("EXAMPLE_LOG_LEVEL"))
	// Output: info
}

func ExampleURL() {
	type Configuration struct {
		PublicURL       url.URL
		ShutdownTimeout time.Duration
		LogLevel        string
	}
	configuration := &Configuration{}

	err := config.LoadConfig([]config.Value{
		config.URL("EXAMPLE_PUBLIC_URL").Schemes("https", "http").Default("https://tasks.example/app").Into(&configuration.PublicURL),
		config.Duration("EXAMPLE_SHUTDOWN_TIMEOUT").Default(10 * time.Second).Into(&configuration.ShutdownTimeout),
		config.Enum("EXAMPLE_LOG_LEVEL", "debug", "info", "warn", "error").Default("info").Into(&configuration.LogLevel),
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(configuration.PublicURL.JoinPath("tasks").String(), configuration.ShutdownTimeout, configuration.LogLevel)
	// Output: https://tasks.example/app/tasks 10s info
}
