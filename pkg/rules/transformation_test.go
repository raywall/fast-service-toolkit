package rules

import (
	"testing"

	"github.com/raywall/fast-service-toolkit/pkg/config"
)

func TestExecuteTransformation(t *testing.T) {
	rm, err := NewRuleManager()
	if err != nil {
		t.Fatalf("Erro ao criar manager: %v", err)
	}

	// CORREÇÃO: Usamos 100.0 para simular float64.
	// O json.Unmarshal (que usaremos na prática) decodifica números como float64 por padrão.
	ctx := map[string]interface{}{
		"input": map[string]interface{}{
			"valor":  100.0, // int(100) causaria erro ao multiplicar por 0.1 (double)
			"status": "vip",
		},
		"vars": map[string]interface{}{},
	}

	tests := []struct {
		name          string
		rule          config.TransformationRule
		expectApplied bool
		expectValue   interface{}
		expectTarget  string
		expectError   bool
	}{
		{
			name: "Sucesso - Condição Verdadeira com Cálculo",
			rule: config.TransformationRule{
				Name:      "calc_desconto",
				Condition: "input.valor >= 100.0", // Comparação segura (double >= double)
				Value:     "input.valor * 0.1",    // double * double = double
				Target:    "vars.desconto",
			},
			expectApplied: true,
			expectValue:   10.0, // CORREÇÃO: Resultado de 100.0 * 0.1 é float64, não int64
			expectTarget:  "vars.desconto",
			expectError:   false,
		},
		{
			name: "Sucesso - Condição Falsa com Else Value",
			rule: config.TransformationRule{
				Name:      "default_taxa",
				Condition: "input.status == 'regular'", // input é vip, então false
				Value:     "15",
				ElseValue: "5",
				Target:    "vars.taxa",
			},
			expectApplied: true,
			expectValue:   int64(5), // Literais inteiros simples no CEL retornam int64
			expectTarget:  "vars.taxa",
			expectError:   false,
		},
		{
			name: "Ignorado - Condição Falsa sem Else",
			rule: config.TransformationRule{
				Name:      "bonus_extra",
				Condition: "input.valor > 1000.0",
				Value:     "500",
				Target:    "vars.bonus",
			},
			expectApplied: false,
			expectError:   false,
		},
		{
			name: "Erro - Condição Inválida",
			rule: config.TransformationRule{
				Name:      "erro_sintaxe",
				Condition: "input.valor > 'texto'", // Comparação inválida
				Value:     "1",
				Target:    "vars.erro",
			},
			expectApplied: false,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := rm.ExecuteTransformation(tt.rule, ctx)

			if tt.expectError {
				if err == nil {
					t.Error("Esperava erro, mas recebeu nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Erro inesperado: %v", err)
			}

			if res.Applied != tt.expectApplied {
				t.Errorf("Applied: esperado %v, recebido %v", tt.expectApplied, res.Applied)
			}

			if tt.expectApplied {
				if res.Value != tt.expectValue {
					t.Errorf("Value: esperado %v (%T), recebido %v (%T)", tt.expectValue, tt.expectValue, res.Value, res.Value)
				}
				if res.Target != tt.expectTarget {
					t.Errorf("Target: esperado %s, recebido %s", tt.expectTarget, res.Target)
				}
			}
		})
	}
}
