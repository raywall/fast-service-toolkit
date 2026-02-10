package responder

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/raywall/fast-service-toolkit/pkg/config"
	"github.com/raywall/fast-service-toolkit/pkg/rules"
)

type ResponseBuilder struct {
	statusCode     int
	bodyTemplate   interface{}
	headerPrograms map[string]cel.Program
	ruleManager    *rules.RuleManager
}

func NewResponseBuilder(cfg config.OutputStep, rm *rules.RuleManager) (*ResponseBuilder, error) {
	cleanBody := sanitize(cfg.Body)

	rb := &ResponseBuilder{
		statusCode:     cfg.StatusCode,
		bodyTemplate:   cleanBody,
		headerPrograms: make(map[string]cel.Program),
		ruleManager:    rm,
	}

	for headerName, rawExpr := range cfg.Headers {
		expr := normalizeExpression(rawExpr)
		prg, err := rm.CompileProgram(expr)
		if err != nil {
			return nil, fmt.Errorf("erro ao compilar header '%s': %w", headerName, err)
		}
		rb.headerPrograms[headerName] = prg
	}

	if err := rb.validateBodyExpressions(rb.bodyTemplate); err != nil {
		return nil, err
	}

	return rb, nil
}

func sanitize(input interface{}) interface{} {
	switch x := input.(type) {
	case map[interface{}]interface{}:
		m := map[string]interface{}{}
		for k, v := range x {
			m[fmt.Sprintf("%v", k)] = sanitize(v)
		}
		return m
	case map[string]interface{}:
		m := map[string]interface{}{}
		for k, v := range x {
			m[k] = sanitize(v)
		}
		return m
	case []interface{}:
		l := make([]interface{}, len(x))
		for i, v := range x {
			l[i] = sanitize(v)
		}
		return l
	default:
		return input
	}
}

func normalizeExpression(expr string) string {
	trimmed := strings.TrimSpace(expr)

	// Caso 1: Variável de Interpolação ${...} -> Remove e retorna o miolo
	if strings.HasPrefix(trimmed, "${") && strings.HasSuffix(trimmed, "}") {
		return trimmed[2 : len(trimmed)-1]
	}

	// Caso 2: Já está entre aspas simples (String Literal CEL) -> Retorna como está
	if strings.HasPrefix(trimmed, "'") && strings.HasSuffix(trimmed, "'") {
		return trimmed
	}

	// Caso 3: Já está entre aspas duplas -> Retorna como está
	if strings.HasPrefix(trimmed, "\"") && strings.HasSuffix(trimmed, "\"") {
		return trimmed
	}

	// Caso 4: Texto puro -> Envolve em aspas simples para virar string CEL
	return fmt.Sprintf("'%s'", trimmed)
}

func (rb *ResponseBuilder) validateBodyExpressions(data interface{}) error {
	switch v := data.(type) {
	case map[string]interface{}:
		for k, val := range v {
			if err := rb.validateBodyExpressions(val); err != nil {
				return fmt.Errorf("campo '%s': %w", k, err)
			}
		}
	case []interface{}:
		for i, val := range v {
			if err := rb.validateBodyExpressions(val); err != nil {
				return fmt.Errorf("item[%d]: %w", i, err)
			}
		}
	case string:
		expr := normalizeExpression(v)
		if _, err := rb.ruleManager.CompileProgram(expr); err != nil {
			return err
		}
	}
	return nil
}

func (rb *ResponseBuilder) Build(ctx map[string]interface{}) (int, []byte, map[string]string, error) {
	processedBody, err := rb.processRecursive(rb.bodyTemplate, ctx)
	if err != nil {
		return 500, nil, nil, err
	}

	bytes, err := json.Marshal(processedBody)
	if err != nil {
		return 500, nil, nil, fmt.Errorf("erro json marshal: %w", err)
	}

	respHeaders := make(map[string]string)
	for name, prg := range rb.headerPrograms {
		out, _, err := prg.Eval(ctx)
		if err != nil {
			return 500, nil, nil, fmt.Errorf("erro eval header '%s': %w", name, err)
		}
		respHeaders[name] = fmt.Sprintf("%v", out.Value())
	}

	return rb.statusCode, bytes, respHeaders, nil
}

func (rb *ResponseBuilder) processRecursive(template interface{}, ctx map[string]interface{}) (interface{}, error) {
	switch v := template.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, val := range v {
			res, err := rb.processRecursive(val, ctx)
			if err != nil {
				return nil, err
			}
			result[k] = res
		}
		return result, nil

	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			res, err := rb.processRecursive(val, ctx)
			if err != nil {
				return nil, err
			}
			result[i] = res
		}
		return result, nil

	case string:
		expr := normalizeExpression(v)
		val, err := rb.ruleManager.EvaluateValue(expr, ctx)
		if err != nil {
			return nil, err
		}
		return val, nil

	default:
		return v, nil
	}
}
