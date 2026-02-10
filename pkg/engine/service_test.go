package engine

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/raywall/fast-service-toolkit/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestServiceEngine_Execute_Flow(t *testing.T) {
	// 1. Configuração Mock
	cfg := &config.ServiceConfig{
		Service: config.ServiceDetails{
			Name:    "test-flow",
			Timeout: "1s",
			Logging: config.LoggingConf{Enabled: false},
			Metrics: config.MetricsConf{Datadog: config.DatadogConf{Enabled: false}},
		},
		Steps: &config.StepsConf{
			Input: config.InputStep{
				Validations: []config.ValidationRule{
					// Mantemos a validação simples
					{
						ID:   "chk_valor",
						Expr: "input.amount > 0",
						OnFail: config.ErrorResponse{
							Code: 400,
							Msg:  "Invalid amount",
						},
					},
				},
			},
			Processing: config.ProcessingStep{
				Transformations: []config.TransformationRule{
					// Multiplica o input por 2.0
					{
						Name:      "double",
						Condition: "true",
						Value:     "input.amount * 2.0",
						Target:    "vars.doubled",
					},
				},
			},
			Output: config.OutputStep{
				StatusCode: 200,
				Body: map[string]interface{}{
					// O output define a chave "result"
					"result": "${vars.doubled}",
				},
			},
		},
	}

	// 2. Inicialização
	svc, err := NewServiceEngine(cfg, "memory")
	if err != nil {
		t.Fatalf("Erro init engine: %v", err)
	}

	// 3. Execução (Sucesso)
	ctx := context.Background()
	// Enviamos um float explícito no JSON
	payload := []byte(`{"amount": 100.0}`)

	code, resp, _, err := svc.Execute(ctx, payload)
	if err != nil {
		t.Fatalf("Erro execute: %v", err)
	}

	if code != 200 {
		t.Errorf("Code esperado 200, recebido %d. Body: %s", code, string(resp))
	}

	var resMap map[string]interface{}
	err = json.Unmarshal(resp, &resMap)
	assert.NoError(t, err)

	// CORREÇÃO: Usamos a chave "result" definida no Output.Body
	// JSON numbers são unmarshaled como float64 por padrão em map[string]interface{}
	expectedValue := 200.0
	actualValue, ok := resMap["result"].(float64)

	if !ok {
		t.Errorf("Esperado float64 em 'result', recebido %T: %v", resMap["result"], resMap["result"])
	} else if actualValue != expectedValue {
		t.Errorf("Transformação falhou. Esperado %v, recebido %v", expectedValue, actualValue)
	}

	// 4. Execução (Falha Input)
	payloadFail := []byte(`{"amount": -10.0}`)
	codeFail, respFail, _, _ := svc.Execute(ctx, payloadFail)

	if codeFail != 400 {
		t.Errorf("Code esperado 400 para erro de input, recebido %d", codeFail)
	}

	expectedErrorJSON := `{"error": "Invalid amount"}`
	assert.JSONEq(t, expectedErrorJSON, string(respFail))
}
