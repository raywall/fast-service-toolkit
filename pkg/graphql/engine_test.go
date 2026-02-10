package graphql

import (
	"context"
	"testing"

	"github.com/raywall/fast-service-toolkit/pkg/config"
	"github.com/raywall/fast-service-toolkit/pkg/rules"
)

func TestNewGraphQLEngine(t *testing.T) {
	// Setup Dependencies
	rm, _ := rules.NewRuleManager()

	// Configuração Mock
	cfg := config.GraphQLConf{
		Enabled: true,
		Route:   "/graphql",
		Types: map[string]config.GQLType{
			"User": {
				Fields: map[string]config.GQLField{
					"id":   {Type: "ID"},
					"name": {Type: "String"},
				},
			},
		},
		Query: map[string]config.GQLField{
			"me": {
				Type: "User",
				Source: &config.EnrichmentSourceConfig{
					Type: "fixed", // Mock source
				},
			},
			"users": {
				Type: "[User]", // Teste de Lista
			},
		},
	}

	// Execução
	engine, err := NewGraphQLEngine(cfg, rm)
	if err != nil {
		t.Fatalf("Erro ao criar engine: %v", err)
	}

	// Validação do Schema
	schema := engine.Schema

	// Verifica se o tipo User foi registrado
	userType := schema.Type("User")
	if userType == nil {
		t.Error("Tipo 'User' não foi criado no schema")
	}

	// Verifica Root Query
	queryType := schema.QueryType()
	if queryType.Fields()["me"] == nil {
		t.Error("Campo 'me' não existe na Query root")
	}

	// Validação de Execução Simples (Introspection)
	// Isso garante que o schema é válido internamente
	query := "{ __schema { types { name } } }"
	res := engine.Execute(context.Background(), query, nil)
	if len(res.Errors) > 0 {
		t.Errorf("Erro na introspecção: %v", res.Errors)
	}
}
