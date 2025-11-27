package main

import (
	"fmt"
	"os"

	"github.com/raywall/dynamodb-quick-service/envloader"
)

func ExampleLoad() {
	// Define sua struct de configuração
	type Config struct {
		Port     string `env:"PORT" envDefault:"8080"`
		Host     string `env:"HOST" envDefault:"localhost"`
		Debug    bool   `env:"DEBUG" envDefault:"false"`
		MaxConns int    `env:"MAX_CONNECTIONS" envDefault:"100"`
	}

	// Define algumas variáveis de ambiente
	os.Setenv("PORT", "9090")
	os.Setenv("DEBUG", "true")

	// Carrega a configuração
	var config Config
	if err := envloader.Load(&config); err != nil {
		panic(err)
	}

	fmt.Printf("Port: %s\n", config.Port)
	fmt.Printf("Host: %s\n", config.Host)
	fmt.Printf("Debug: %t\n", config.Debug)
	fmt.Printf("MaxConns: %d\n", config.MaxConns)

	// Output:
	// Port: 9090
	// Host: localhost
	// Debug: true
	// MaxConns: 100
}

func ExampleLoad_complex() {
	type DatabaseConfig struct {
		Host string `env:"DB_HOST" envDefault:"localhost"`
		Port int    `env:"DB_PORT" envDefault:"5432"`
		Name string `env:"DB_NAME" envDefault:"mydb"`
	}

	type ServerConfig struct {
		Port    int    `env:"SERVER_PORT" envDefault:"8080"`
		Timeout int    `env:"TIMEOUT" envDefault:"30"`
		Env     string `env:"ENVIRONMENT" envDefault:"production"`
	}

	type AppConfig struct {
		Server   ServerConfig
		Database DatabaseConfig
		AppName  string `env:"APP_NAME" envDefault:"MyApp"`
	}

	// Carrega com valores padrão
	var config AppConfig
	envloader.MustLoad(&config)

	fmt.Printf("App: %s\n", config.AppName)
	fmt.Printf("Server Port: %d\n", config.Server.Port)
	fmt.Printf("Database: %s:%d/%s\n", config.Database.Host, config.Database.Port, config.Database.Name)

	// Output:
	// App: MyApp
	// Server Port: 8080
	// Database: localhost:5432/mydb
}

func main() {
	ExampleLoad()
	ExampleLoad_complex()
}