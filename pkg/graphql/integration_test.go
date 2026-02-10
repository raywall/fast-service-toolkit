package graphql

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/raywall/fast-service-lab/pkg/config"
	"github.com/raywall/fast-service-lab/pkg/enrichment"
	"github.com/raywall/fast-service-lab/pkg/rules"
)

// MockHttpClient para interceptar chamadas REST dentro do Resolver
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

func TestGraphQL_Mesh_ComplexFlow(t *testing.T) {
	// 1. Setup Rule Manager
	rm, _ := rules.NewRuleManager()

	// 2. Setup Servidor REST Mockado (Simula API de Usuários)
	userServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Valida se o Token de Auth foi propagado
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer my-jwt-token" {
			http.Error(w, "Unauthorized", 401)
			return
		}

		// Retorna usuário baseado no ID da query string
		userID := r.URL.Query().Get("id")
		w.Header().Set("Content-Type", "application/json")

		if userID == "101" {
			w.Write([]byte(`{"id": 101, "name": "Alice"}`))
		} else {
			w.Write([]byte(`{"id": 999, "name": "Unknown"}`))
		}
	}))
	defer userServer.Close()

	// Injeta o cliente HTTP do servidor de teste no pacote enrichment
	// IMPORTANTE: Isso assume que enrichment.Client é exportado e modificável (conforme visto em sources.go)
	originalClient := enrichment.Client
	enrichment.Client = userServer.Client()
	defer func() { enrichment.Client = originalClient }()

	// 3. Configuração do Schema (Blog: Posts -> Author)
	cfg := config.GraphQLConf{
		Enabled: true,
		Types: map[string]config.GQLType{
			"Post": {
				Fields: map[string]config.GQLField{
					"title":    {Type: "String"},
					"authorId": {Type: "String"}, // Campo interno para link
					"author": {
						Type: "User",
						Source: &config.EnrichmentSourceConfig{
							Type: "rest",
							Params: map[string]interface{}{
								"method": "GET",
								// Usa o authorId do objeto pai (source) para buscar na API
								"url": "'" + userServer.URL + "?id=' + string(source.authorId)",
							},
							Headers: map[string]string{
								// Usa o token do contexto de auth
								"Authorization": "'Bearer ' + string(auth.token)",
							},
						},
					},
				},
			},
			"User": {
				Fields: map[string]config.GQLField{
					"name": {Type: "String"},
				},
			},
		},
		Query: map[string]config.GQLField{
			"feed": {
				Type: "[Post]",
				Source: &config.EnrichmentSourceConfig{
					Type: "fixed",
					Params: map[string]interface{}{
						"value": []interface{}{
							map[string]interface{}{"title": "Hello World", "authorId": "101"},
							map[string]interface{}{"title": "Go is great", "authorId": "102"},
						},
					},
				},
			},
		},
	}

	// 4. Inicializa Engine
	engine, err := NewGraphQLEngine(cfg, rm)
	if err != nil {
		t.Fatalf("Erro init engine: %v", err)
	}

	// 5. Prepara Contexto com Auth
	ctx := context.Background()
	authCtx := map[string]interface{}{
		"token": "my-jwt-token",
	}
	ctx = context.WithValue(ctx, "auth_context", authCtx)

	// 6. Executa Query
	query := `
		query {
			feed {
				title
				author {
					name
				}
			}
		}
	`
	result := engine.Execute(ctx, query, nil)

	// 7. Validações
	if len(result.Errors) > 0 {
		t.Fatalf("Erros na execução GraphQL: %v", result.Errors)
	}

	data := result.Data.(map[string]interface{})
	feed := data["feed"].([]interface{})

	// Post 1 (Alice)
	post1 := feed[0].(map[string]interface{})
	author1 := post1["author"].(map[string]interface{})
	if author1["name"] != "Alice" {
		t.Errorf("Esperado autor Alice, recebido %v", author1["name"])
	}

	// Post 2 (Unknown)
	post2 := feed[1].(map[string]interface{})
	author2 := post2["author"].(map[string]interface{})
	if author2["name"] != "Unknown" {
		t.Errorf("Esperado autor Unknown, recebido %v", author2["name"])
	}
}
