package rules

import (
	"fmt"

	"github.com/raywall/fast-service-lab/pkg/config"
)

// TransformationResult contém o resultado de uma operação de transformação.
type TransformationResult struct {
	Applied bool
	Target  string
	Value   interface{}
}

// ExecuteTransformation processa uma regra de transformação completa.
// Verifica a condição e, se atendida, calcula o valor. Se não, verifica o ElseValue.
func (rm *RuleManager) ExecuteTransformation(rule config.TransformationRule, ctx map[string]interface{}) (*TransformationResult, error) {
	// 1. Avaliar a Condição (Deve retornar booleano)
	conditionMet, err := rm.EvaluateBool(rule.Condition, ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao avaliar condição da transformação '%s': %w", rule.Name, err)
	}

	var exprToEvaluate string

	if conditionMet {
		exprToEvaluate = rule.Value
	} else {
		// Se a condição falhou e existe um ElseValue, usamos ele
		if rule.ElseValue != "" {
			exprToEvaluate = rule.ElseValue
		} else {
			// Nenhuma ação necessária (condição falsa e sem else)
			return &TransformationResult{Applied: false}, nil
		}
	}

	// 2. Calcular o Valor Final
	val, err := rm.EvaluateValue(exprToEvaluate, ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao calcular valor da transformação '%s': %w", rule.Name, err)
	}

	return &TransformationResult{
		Target:  rule.Target,
		Value:   val,
		Applied: true,
	}, nil
}
