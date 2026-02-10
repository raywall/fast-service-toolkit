package graphql

import (
	"context"
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/raywall/fast-service-toolkit/pkg/config"
	"github.com/raywall/fast-service-toolkit/pkg/enrichment"
	"github.com/raywall/fast-service-toolkit/pkg/rules"
)

// makeResolver cria uma função de resolução ASSÍNCRONA para concorrência
func makeResolver(fieldDef config.GQLField, rm *rules.RuleManager) graphql.FieldResolveFn {
	// TRAVA DE SEGURANÇA 1: Validação no Build Time
	if rm == nil {
		fmt.Printf("❌ FATAL: RuleManager é NIL na criação do resolver para: %+v\n", fieldDef)
	}

	return func(p graphql.ResolveParams) (interface{}, error) {
		resultChan := make(chan interface{})

		var authData interface{}
		if p.Context != nil {
			authData = p.Context.Value("auth_context")
		}

		celCtx := map[string]interface{}{
			"args":   p.Args,
			"source": p.Source,
			"auth":   authData,
		}

		// Garante captura por valor do ponteiro rm
		go func(localRm *rules.RuleManager) {
			defer close(resultChan)

			// TRAVA DE SEGURANÇA 2: Validação no Runtime
			if localRm == nil {
				fmt.Println("❌ PANIC EVITED: RuleManager nulo dentro da goroutine!")
				resultChan <- fmt.Errorf("erro interno: RuleManager nulo")
				return
			}

			resolvedParams, err := resolveParams(fieldDef.Source.Params, celCtx, localRm)
			if err != nil {
				resultChan <- err
				return
			}

			var headers map[string]string
			if fieldDef.Source.Headers != nil {
				h, err := resolveHeaders(fieldDef.Source.Headers, celCtx, localRm)
				if err != nil {
					resultChan <- err
					return
				}
				headers = h
			}

			ctxIO := p.Context
			if ctxIO == nil {
				ctxIO = context.Background()
			}

			var res interface{}
			var execErr error

			switch fieldDef.Source.Type {
			case "fixed":
				res, execErr = enrichment.ProcessFixed(resolvedParams)
			case "rest":
				res, execErr = enrichment.ProcessRest(ctxIO, toString(resolvedParams["method"]), toString(resolvedParams["url"]), headers, resolvedParams["body"])
			case "dynamodb":
				keyMap := toMap(resolvedParams["key"])
				if len(keyMap) == 0 {
					execErr = fmt.Errorf("chave vazia")
				} else {
					res, execErr = enrichment.ProcessDynamoDB(ctxIO, toString(resolvedParams["region"]), toString(resolvedParams["table"]), keyMap)
				}
			default:
				execErr = fmt.Errorf("adapter desconhecido: %s", fieldDef.Source.Type)
			}

			if execErr != nil {
				resultChan <- execErr
			} else {
				resultChan <- res
			}
		}(rm)

		return func() (interface{}, error) {
			val := <-resultChan
			if err, ok := val.(error); ok {
				return nil, err
			}
			return val, nil
		}, nil
	}
}

// Funções Auxiliares

func resolveParams(raw map[string]interface{}, ctx map[string]interface{}, rm *rules.RuleManager) (map[string]interface{}, error) {
	resolved := make(map[string]interface{})
	for k, v := range raw {
		res, err := resolveRecursive(v, ctx, rm)
		if err != nil {
			return nil, err
		}
		resolved[k] = res
	}
	return resolved, nil
}

func resolveHeaders(raw map[string]string, ctx map[string]interface{}, rm *rules.RuleManager) (map[string]string, error) {
	resolved := make(map[string]string)
	for k, v := range raw {
		// Tenta avaliar como CEL
		res, err := rm.EvaluateValue(v, ctx)
		if err != nil {
			// CORREÇÃO: Se falhar (ex: "Fixo" não é variável), usa o valor original como string estática
			resolved[k] = v
		} else {
			resolved[k] = fmt.Sprintf("%v", res)
		}
	}
	return resolved, nil
}

func resolveRecursive(val interface{}, ctx map[string]interface{}, rm *rules.RuleManager) (interface{}, error) {
	switch v := val.(type) {
	case string:
		out, err := rm.EvaluateValue(v, ctx)
		if err != nil {
			// Fallback para string estática
			return v, nil
		}
		return out, nil

	case map[string]interface{}:
		newMap := make(map[string]interface{})
		for mk, mv := range v {
			res, err := resolveRecursive(mv, ctx, rm)
			if err != nil {
				return nil, err
			}
			newMap[mk] = res
		}
		return newMap, nil

	case map[interface{}]interface{}:
		newMap := make(map[string]interface{})
		for mk, mv := range v {
			res, err := resolveRecursive(mv, ctx, rm)
			if err != nil {
				return nil, err
			}
			newMap[fmt.Sprintf("%v", mk)] = res
		}
		return newMap, nil

	case []interface{}:
		newList := make([]interface{}, len(v))
		for i, item := range v {
			res, err := resolveRecursive(item, ctx, rm)
			if err != nil {
				return nil, err
			}
			newList[i] = res
		}
		return newList, nil

	default:
		return v, nil
	}
}

func getEnvVars() map[string]string {
	return map[string]string{}
}

func toMap(v interface{}) map[string]interface{} {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	if mGen, ok := v.(map[interface{}]interface{}); ok {
		out := make(map[string]interface{})
		for k, val := range mGen {
			out[fmt.Sprintf("%v", k)] = val
		}
		return out
	}
	return nil
}

func toString(v interface{}) string {
	return fmt.Sprintf("%v", v)
}
