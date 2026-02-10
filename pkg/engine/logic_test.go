package engine

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/raywall/fast-service-lab/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestBusinessLogic_ComplexFlow(t *testing.T) {
	// Configuração do Cenário
	cfg := &config.ServiceConfig{
		Service: config.ServiceDetails{
			Name:    "logic-test",
			Timeout: "1s",
			Logging: config.LoggingConf{Enabled: false},
		},
		// CORRIGIDO: Ponteiro &StepsConf
		Steps: &config.StepsConf{
			Input: config.InputStep{
				Validations: []config.ValidationRule{
					{
						ID:     "val_age",
						Expr:   "int(input.age) >= 18",
						OnFail: config.ErrorResponse{Code: 400, Msg: "Menor de idade"},
					},
				},
			},
			Processing: config.ProcessingStep{
				Transformations: []config.TransformationRule{
					{
						Name:      "calc_discount",
						Target:    "vars.discount",
						Condition: "input.vip == true",
						Value:     "10.0",
						ElseValue: "0.0",
					},
				},
			},
			Output: config.OutputStep{
				StatusCode: 200,
				// Body simples
				Body: map[string]interface{}{
					"status":      "APPROVED", // normalizeExpression vai tratar isso
					"final_price": "${input.price - vars.discount}",
				},
			},
		},
	}

	// Inicializa Engine
	eng, err := NewServiceEngine(cfg, "memory")
	assert.NoError(t, err)

	// Caso de Sucesso
	payload := []byte(`{"age": 25, "vip": true, "price": 100.0}`)
	code, resp, _, err := eng.Execute(context.Background(), payload)

	assert.NoError(t, err)
	assert.Equal(t, 200, code)

	var respMap map[string]interface{}
	json.Unmarshal(resp, &respMap)
	assert.Equal(t, "APPROVED", respMap["status"])
	assert.Equal(t, 90.0, respMap["final_price"])
}
