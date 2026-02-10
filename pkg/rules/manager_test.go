package rules

import (
	"testing"
)

func TestEvaluateBool(t *testing.T) {
	rm, _ := NewRuleManager()

	data := map[string]interface{}{
		"input": map[string]interface{}{"age": 20, "type": "admin"},
	}

	// Cenário 1: Sucesso
	ok, err := rm.EvaluateBool("input.age >= 18 && input.type == 'admin'", data)
	if err != nil || !ok {
		t.Errorf("Falha na validação correta: %v", err)
	}

	// Cenário 2: Falha
	ok, _ = rm.EvaluateBool("input.age < 10", data)
	if ok {
		t.Error("Deveria retornar false")
	}
}

func TestEvaluateValue(t *testing.T) {
	rm, _ := NewRuleManager()
	data := map[string]interface{}{
		"input": map[string]interface{}{"val": 100},
	}

	res, err := rm.EvaluateValue("input.val * 2", data)
	if err != nil {
		t.Fatalf("Erro ao avaliar: %v", err)
	}

	if res.(int64) != 200 {
		t.Errorf("Esperado 200, recebido %v", res)
	}
}
