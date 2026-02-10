package engine

import (
	"context"
	"testing"
	"time"

	"github.com/raywall/fast-service-toolkit/pkg/auth"
	"github.com/raywall/fast-service-toolkit/pkg/config"
	"github.com/raywall/fast-service-toolkit/pkg/rules" // Importante para inicializar o RuleManager
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// --- Mocks e Helpers ---

// mockTokenFetcher cria um fetcher que retorna um token fixo sem rede
func mockTokenFetcher(token string) auth.TokenFetcher {
	return func(ctx context.Context) (string, time.Duration, error) {
		return token, 1 * time.Hour, nil
	}
}

// --- Testes ---

func TestMiddleware_Enrichment_Execution(t *testing.T) {
	// 1. Configuração do Middleware (Source Fixed)
	mwConf := config.MiddlewareConf{
		Type: "enrichment",
		ID:   "enrich_1",
		Config: map[string]interface{}{
			"strategy": "parallel",
			"sources": []interface{}{
				map[string]interface{}{
					"name": "mock_data",
					"type": "fixed",
					"params": map[string]interface{}{
						"value": map[string]interface{}{
							"user_level": "gold",
							"active":     true,
						},
					},
				},
			},
		},
	}

	// 2. Setup Engine
	// CORREÇÃO: Precisamos inicializar o RuleManager, pois o enrichment usa interpolação
	rm, err := rules.NewRuleManager()
	assert.NoError(t, err)

	se := &ServiceEngine{
		Logger:      zerolog.Nop(),
		RuleManager: rm, // Injeção obrigatória
	}

	// Contexto de execução simulado
	execCtx := map[string]interface{}{
		"detection": make(map[string]interface{}),
		"vars":      make(map[string]interface{}),
		"env":       map[string]string{}, // Necessário para interpolação
	}

	// 3. Execução
	err = se.executeEnrichmentMiddleware(context.Background(), mwConf, execCtx)
	assert.NoError(t, err)

	// 4. Asserts
	detection := execCtx["detection"].(map[string]interface{})
	assert.Contains(t, detection, "mock_data")

	data := detection["mock_data"].(map[string]interface{})
	assert.Equal(t, "gold", data["user_level"])
	assert.Equal(t, true, data["active"])
}

func TestMiddleware_Auth_Injection(t *testing.T) {
	// 1. Setup Auth Manager com Mock
	mockMgr := auth.NewManager(mockTokenFetcher("mocked-jwt-token-123"))
	err := mockMgr.Start(context.Background())
	assert.NoError(t, err)
	defer mockMgr.Stop()

	// 2. Setup Engine
	// Nota: Em testes unitários de middleware, não precisamos de Config.Steps validos,
	// apenas a configuração do middleware sendo testado.
	se := &ServiceEngine{
		Config: &config.ServiceConfig{
			Middlewares: []config.MiddlewareConf{
				{
					Type: "auth_provider",
					ID:   "auth_partners",
					Config: map[string]interface{}{
						"output_var": "partner_token",
					},
				},
			},
		},
		AuthManagers: map[string]*auth.Manager{
			"auth_partners": mockMgr,
		},
	}

	// 3. Execução (RunMiddlewares)
	ctx, err := se.RunMiddlewares(context.Background())
	assert.NoError(t, err)

	// 4. Asserts
	authCtxVal := ctx.Value("auth_context")
	assert.NotNil(t, authCtxVal, "Contexto de auth não deve ser nulo")

	authMap, ok := authCtxVal.(map[string]interface{})
	assert.True(t, ok, "Tipo do contexto de auth inválido")

	// CORREÇÃO: O token agora fica dentro do namespace do ID do middleware
	// authMap["auth_partners"]["partner_token"]

	partnersScope, ok := authMap["auth_partners"].(map[string]interface{})
	assert.True(t, ok, "Namespace 'auth_partners' não encontrado no contexto")

	assert.Equal(t, "mocked-jwt-token-123", partnersScope["partner_token"])
}
