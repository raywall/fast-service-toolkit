package injector_test

import (
	"context"
	"os"
	"testing"

	"github.com/raywall/fast-service-lab/pkg/config/injector"
	"github.com/stretchr/testify/assert"
)

type TestConfig struct {
	Name        string                 `yaml:"name" env:"SERVICE_NAME"` // Caso 1: Tag
	APIKey      string                 `yaml:"api_key"`                 // Caso 2: Interpolação String "${env.KEY}"
	Description string                 `yaml:"description"`             // Caso 3: Texto misto "Service running in ${env.REGION}"
	Meta        map[string]interface{} // Caso 4: Map Dinâmico
	Nested      *NestedConfig
}

type NestedConfig struct {
	URL string
}

func TestInjector_Inject_Environment(t *testing.T) {
	// Setup Environment
	os.Setenv("SERVICE_NAME", "OrderService")
	os.Setenv("API_KEY", "12345-abcde")
	os.Setenv("REGION", "us-east-1")
	os.Setenv("DB_HOST", "localhost")

	inj := injector.New()

	target := &TestConfig{
		Name:        "Placeholder", // Deve ser sobrescrito pela tag
		APIKey:      "${env.API_KEY}",
		Description: "Service running in ${env.REGION}",
		Meta: map[string]interface{}{
			"db_host": "${env.DB_HOST}",
			"timeout": 5000, // Inteiro não deve ser tocado
		},
		Nested: &NestedConfig{
			URL: "https://${env.REGION}.api.com",
		},
	}

	err := inj.Inject(context.Background(), target)
	assert.NoError(t, err)

	// Asserts
	assert.Equal(t, "OrderService", target.Name, "Tag env não funcionou")
	assert.Equal(t, "12345-abcde", target.APIKey, "Interpolação direta falhou")
	assert.Equal(t, "Service running in us-east-1", target.Description, "Interpolação mista falhou")
	assert.Equal(t, "localhost", target.Meta["db_host"], "Interpolação em mapa falhou")
	assert.Equal(t, "https://us-east-1.api.com", target.Nested.URL, "Interpolação aninhada falhou")
}
