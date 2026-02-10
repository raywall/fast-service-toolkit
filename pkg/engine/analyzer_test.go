package engine

import (
	"testing"

	"github.com/raywall/fast-service-lab/pkg/config"
)

func TestAnalyze_Detection(t *testing.T) {
	// Config com erro de sintaxe CEL intencional
	cfg := &config.ServiceConfig{
		Steps: &config.StepsConf{
			Input: config.InputStep{
				Validations: []config.ValidationRule{
					{ID: "rule1", Expr: "input.valor > "}, // ERRO: Expressão incompleta
				},
			},
			Processing: config.ProcessingStep{
				Transformations: []config.TransformationRule{
					{Name: "t1", Condition: "true", Value: "10 *", Target: "vars.x"}, // ERRO
				},
			},
			Output: config.OutputStep{
				Body: map[string]interface{}{"result": "vars.x"},
			},
		},
	}

	report, err := Analyze(cfg)
	if err != nil {
		t.Fatalf("Erro inesperado: %v", err)
	}

	if report.Valid {
		t.Error("Deveria ser inválido devido aos erros de sintaxe CEL")
	}

	if len(report.Errors) < 2 {
		t.Errorf("Esperado pelo menos 2 erros, encontrados %d", len(report.Errors))
	}
}
