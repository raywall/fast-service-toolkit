// Copyright 2025 Raywall Malheiros de Souza
// Licensed under the Mozilla Public License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.mozilla.org/en-US/MPL/2.0/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package envloader

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_StringFields(t *testing.T) {
	type Config struct {
		Port     string `env:"PORT" default:"8080"`
		Host     string `env:"HOST" default:"localhost"`
		LogLevel string `env:"LOG_LEVEL" default:"info"`
	}

	// Test with default values
	config := &Config{}
	err := Load(config)
	require.NoError(t, err)

	assert.Equal(t, "8080", config.Port)
	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, "info", config.LogLevel)

	// Test with environment variables
	os.Setenv("PORT", "9090")
	os.Setenv("HOST", "127.0.0.1")
	os.Setenv("LOG_LEVEL", "debug")

	config2 := &Config{}
	err = Load(config2)
	require.NoError(t, err)

	assert.Equal(t, "9090", config2.Port)
	assert.Equal(t, "127.0.0.1", config2.Host)
	assert.Equal(t, "debug", config2.LogLevel)

	// Cleanup
	os.Unsetenv("PORT")
	os.Unsetenv("HOST")
	os.Unsetenv("LOG_LEVEL")
}

func TestLoad_NumericFields(t *testing.T) {
	type Config struct {
		Port        int    `env:"PORT" default:"8080"`
		MaxConn     int32  `env:"MAX_CONNECTIONS" default:"100"`
		Timeout     int64  `env:"TIMEOUT" default:"30"`
		MaxFileSize uint64 `env:"MAX_FILE_SIZE" default:"1048576"`
	}

	config := &Config{}
	err := Load(config)
	require.NoError(t, err)

	assert.Equal(t, 8080, config.Port)
	assert.Equal(t, int32(100), config.MaxConn)
	assert.Equal(t, int64(30), config.Timeout)
	assert.Equal(t, uint64(1048576), config.MaxFileSize)

	// Test with environment variables
	os.Setenv("PORT", "9090")
	os.Setenv("MAX_CONNECTIONS", "500")

	config2 := &Config{}
	err = Load(config2)
	require.NoError(t, err)

	assert.Equal(t, 9090, config2.Port)
	assert.Equal(t, int32(500), config2.MaxConn)

	// Cleanup
	os.Unsetenv("PORT")
	os.Unsetenv("MAX_CONNECTIONS")
}

func TestLoad_BoolFields(t *testing.T) {
	type Config struct {
		Debug    bool `env:"DEBUG" default:"true"`
		Enabled  bool `env:"ENABLED" default:"false"`
		FeatureX bool `env:"FEATURE_X" default:"1"`
		FeatureY bool `env:"FEATURE_Y" default:"0"`
	}

	config := &Config{}
	err := Load(config)
	require.NoError(t, err)

	assert.True(t, config.Debug)
	assert.False(t, config.Enabled)
	assert.True(t, config.FeatureX)
	assert.False(t, config.FeatureY)

	// Test with environment variables
	os.Setenv("DEBUG", "false")
	os.Setenv("ENABLED", "true")

	config2 := &Config{}
	err = Load(config2)
	require.NoError(t, err)

	assert.False(t, config2.Debug)
	assert.True(t, config2.Enabled)

	// Cleanup
	os.Unsetenv("DEBUG")
	os.Unsetenv("ENABLED")
}

func TestLoad_FloatFields(t *testing.T) {
	type Config struct {
		Ratio    float32 `env:"RATIO" default:"1.5"`
		Price    float64 `env:"PRICE" default:"99.99"`
		Discount float64 `env:"DISCOUNT" default:"0.1"`
	}

	config := &Config{}
	err := Load(config)
	require.NoError(t, err)

	assert.Equal(t, float32(1.5), config.Ratio)
	assert.Equal(t, 99.99, config.Price)
	assert.Equal(t, 0.1, config.Discount)

	// Test with environment variables
	os.Setenv("PRICE", "149.99")

	config2 := &Config{}
	err = Load(config2)
	require.NoError(t, err)

	assert.Equal(t, 149.99, config2.Price)

	// Cleanup
	os.Unsetenv("PRICE")
}

func TestLoad_WithoutEnvTag(t *testing.T) {
	type Config struct {
		Port     string `env:"PORT" default:"8080"`
		Host     string // Sem tag env - deve ser ignorado
		LogLevel string `env:"LOG_LEVEL" default:"info"`
	}

	config := &Config{
		Host: "original", // Valor original deve ser mantido
	}
	err := Load(config)
	require.NoError(t, err)

	assert.Equal(t, "8080", config.Port)
	assert.Equal(t, "original", config.Host) // Não foi alterado
	assert.Equal(t, "info", config.LogLevel)
}

func TestLoad_EmptyEnvVar(t *testing.T) {
	type Config struct {
		Port    string `env:"PORT" default:"8080"`
		Timeout string `env:"TIMEOUT"` // Sem default - deve ficar vazio
	}

	config := &Config{}
	err := Load(config)
	require.NoError(t, err)

	assert.Equal(t, "8080", config.Port)
	assert.Equal(t, "", config.Timeout)
}

func TestLoad_EnvironmentOverridesDefault(t *testing.T) {
	type Config struct {
		Port string `env:"PORT" default:"8080"`
	}

	os.Setenv("PORT", "9090")

	config := &Config{}
	err := Load(config)
	require.NoError(t, err)

	assert.Equal(t, "9090", config.Port) // Environment tem prioridade

	// Cleanup
	os.Unsetenv("PORT")
}

func TestLoad_InvalidConfig(t *testing.T) {
	// Não é um ponteiro
	var config string
	err := Load(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pointer to struct")

	// Não é uma struct
	var config2 int
	err = Load(&config2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pointer to struct")
}

func TestLoad_ConversionErrors(t *testing.T) {
	type Config struct {
		Port int `env:"PORT" default:"not-a-number"`
	}

	config := &Config{}
	err := Load(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error setting field Port")
}

func TestMustLoad(t *testing.T) {
	type Config struct {
		Port string `env:"PORT" default:"8080"`
	}

	// Deve funcionar sem panic
	config := &Config{}
	assert.NotPanics(t, func() {
		MustLoad(config)
	})
	assert.Equal(t, "8080", config.Port)

	// Deve dar panic com config inválido
	assert.Panics(t, func() {
		MustLoad("not-a-pointer")
	})
}

func TestLoad_ComplexStruct(t *testing.T) {
	type DatabaseConfig struct {
		Host     string `env:"DB_HOST" default:"localhost"`
		Port     int    `env:"DB_PORT" default:"5432"`
		Username string `env:"DB_USER" default:"postgres"`
		Password string `env:"DB_PASS" default:""`
		SSLMode  bool   `env:"DB_SSL" default:"true"`
	}

	type ServerConfig struct {
		Port        int    `env:"SERVER_PORT" default:"8080"`
		Host        string `env:"SERVER_HOST" default:"0.0.0.0"`
		Debug       bool   `env:"DEBUG" default:"false"`
		Environment string `env:"ENV" default:"production"`
	}

	type AppConfig struct {
		Server   ServerConfig
		Database DatabaseConfig
		Name     string `env:"APP_NAME" default:"MyApp"`
		Version  string `env:"APP_VERSION" default:"1.0.0"`
	}

	// Set some environment variables
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("DB_HOST", "db.example.com")
	os.Setenv("DEBUG", "true")

	config := &AppConfig{}
	err := Load(config)
	require.NoError(t, err)

	// Verifica valores
	assert.Equal(t, "MyApp", config.Name)
	assert.Equal(t, "1.0.0", config.Version)

	// Server config
	assert.Equal(t, 9090, config.Server.Port)                // Do environment
	assert.Equal(t, "0.0.0.0", config.Server.Host)           // Default
	assert.True(t, config.Server.Debug)                      // Do environment
	assert.Equal(t, "production", config.Server.Environment) // Default

	// Database config
	assert.Equal(t, "db.example.com", config.Database.Host) // Do environment
	assert.Equal(t, 5432, config.Database.Port)             // Default
	assert.Equal(t, "postgres", config.Database.Username)   // Default
	assert.Equal(t, "", config.Database.Password)           // Default vazio
	assert.True(t, config.Database.SSLMode)                 // Default

	// Cleanup
	os.Unsetenv("SERVER_PORT")
	os.Unsetenv("DB_HOST")
	os.Unsetenv("DEBUG")
}
