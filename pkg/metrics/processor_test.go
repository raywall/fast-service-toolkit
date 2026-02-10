package metrics

import (
	"testing"

	"github.com/raywall/fast-service-toolkit/pkg/config"
	"github.com/raywall/fast-service-toolkit/pkg/rules"
)

// MockProvider para verificar chamadas
type MockProvider struct {
	LastCallType string
	LastName     string
	LastValue    float64
	LastTags     []string
}

func (m *MockProvider) Count(name string, val float64, tags []string) error {
	m.LastCallType = "count"
	m.LastName = name
	m.LastValue = val
	m.LastTags = tags
	return nil
}
func (m *MockProvider) Gauge(name string, val float64, tags []string) error {
	m.LastCallType = "gauge"
	m.LastName = name
	m.LastValue = val
	m.LastTags = tags
	return nil
}
func (m *MockProvider) Histogram(name string, val float64, tags []string) error {
	return nil
}

func TestProcessor_ProcessRules(t *testing.T) {
	// Setup
	rm, _ := rules.NewRuleManager()
	provider := &MockProvider{}

	// Definições (Configuração global)
	defs := []config.CustomMetricDefinition{
		{ID: "transacao_ok", Name: "app.transaction.success", Type: "count"},
		{ID: "tempo_proc", Name: "app.processing.time", Type: "gauge"},
	}

	processor := NewProcessor(defs, provider, rm)

	// Contexto de execução (Dados)
	ctx := map[string]interface{}{
		"input": map[string]interface{}{"moeda": "BRL"},
		"vars":  map[string]interface{}{"duration": 150},
	}

	t.Run("Deve registrar Count com tags dinâmicas", func(t *testing.T) {
		rulesList := []config.MetricRegistrationRule{
			{
				MetricID: "transacao_ok",
				Value:    "1", // Valor fixo
				Tags: map[string]string{
					"currency": "input.moeda", // Tag dinâmica
					"env":      "'prod'",      // Tag fixa string
				},
			},
		}

		err := processor.ProcessRules(rulesList, ctx)
		if err != nil {
			t.Fatalf("Erro inesperado: %v", err)
		}

		if provider.LastCallType != "count" {
			t.Errorf("Esperado count, recebido %s", provider.LastCallType)
		}
		if provider.LastName != "app.transaction.success" {
			t.Errorf("Nome incorreto: %s", provider.LastName)
		}
		if provider.LastValue != 1.0 {
			t.Errorf("Valor incorreto: %f", provider.LastValue)
		}

		// Verificação simples de tags (a ordem do map não é garantida, então verificamos existência)
		tagFound := false
		for _, tag := range provider.LastTags {
			if tag == "currency:BRL" {
				tagFound = true
			}
		}
		if !tagFound {
			t.Errorf("Tag dinâmica currency:BRL não encontrada nas tags: %v", provider.LastTags)
		}
	})

	t.Run("Deve registrar Gauge com valor dinâmico", func(t *testing.T) {
		rulesList := []config.MetricRegistrationRule{
			{
				MetricID: "tempo_proc",
				Value:    "vars.duration", // Valor vindo de variável
				Tags:     nil,
			},
		}

		err := processor.ProcessRules(rulesList, ctx)
		if err != nil {
			t.Fatalf("Erro: %v", err)
		}

		if provider.LastCallType != "gauge" {
			t.Errorf("Esperado gauge, recebido %s", provider.LastCallType)
		}
		if provider.LastValue != 150.0 {
			t.Errorf("Valor esperado 150.0, recebido %f", provider.LastValue)
		}
	})
}
