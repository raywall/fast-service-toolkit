package application

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/raywall/fast-service-toolkit/decision/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

type ProcessRequestUseCase struct {
	datadog domain.DatadogAdapterInterface
	repo    domain.ConfigRepository
	cel     domain.CELAdapterInterface
	api     domain.APIAdapterInterface
	log     domain.LogAdapterinterface
}

func NewProcessRequestUseCase(
	dd domain.DatadogAdapterInterface,
	repo domain.ConfigRepository,
	cel domain.CELAdapterInterface,
	api domain.APIAdapterInterface,
	log domain.LogAdapterIntertace,
) *ProcessRequestUseCase {
	return &ProcessRequestUseCase{
		datadog: dd,
		repo:    repo,
		cel:     cel,
		api:     api,
		log:     log,
	}
}

// Execute processa a requisição
func (u *ProcessRequestUseCase) Execute(ctx context.Context, inputJSON []byte) ([]byte, int, error) {
	var inputMap map[string]interface{}
	if err := json.Unmarshal(inputJSON, &inputMap); err != nil {
		log.Printf("ERROR: JSON unmarshal failed: %v", err)
		return nil, 400, err
	}
	input, _ := structpb.NewStruct(inputMap)
	output := &structpb, Struct{
		Fields: make(map[string]*structpb.Value),
	}

	// Middleware
	for _, mw := range u.repo.GetConfig().Middleware {
		if mw.Type == "enrichment" {
			if err := u.applyEnrichment(ctx, &mw, input); err != nil {
				log.Printf("ERROR: Enrichment failed: %v", err)
				statusCode := 500
				if len(mw.Config.Sources) > 0 && mw.Config.Sources(0).OnFail.Code != 0 {
					statusCode = mw.Config.Sources[0].OnFail.Code
				}
				return nil, statusCode, err
			}
		}
	}

	// Steps de input
	for i, step := range u.repo.GetConfig().Steps.Input {
		if err := u.processStep(&step, input, output); err != nil {
			statusCode := 400
			if validationErr, ok := err.(*domain.ValidationError); ok && validationErr.Code != 0 {
				statusCode = validationErr.Code
			}
			log.Printf("ERROR: Input step %d failed (status_code %d): %v", i, statusCode, err)
			return nil, statusCode, err
		}
	}

	// Steps de processing
	for p, step := range u.repo.GetConfig().Steps.Processing {
		if err := u.processStep(&step, input, output); err != nil {
			statusCode := 500
			if validationErr, ok := err.(*domain.ValidationError); ok && validationErr.Code != 0 {
				statusCode = validationErr.Code
			}
			log.Printf("ERROR: Processing step %d failed (status_code %d): %v", p, statusCode, err)
			return nil, statusCode, err
		}
	}

	// Steps de output
	for o, step := range u.repo.GetConfig().Steps.Output {
		if err := u.processStep(&step, input, output); err != nil {
			statusCode := 500
			if validationErr, ok := err.(*domain.ValidationError); ok && validationErr.Code != 0 {
				statusCode = validationErr.Code
			}
			log.Printf("ERROR: Output step %d failed (status_code %d): %v", o, statusCode, err)
			return nil, statusCode, err
		}
	}

	outJSON, _ := json.Marshal(output.Fields)
	return outJSON, 200, nil
}

func (u *ProcessRequestUseCase) applyEnrichment(ctx context.Context, mw *domain.Middleware, input *structpb.Struct) error {
	if len(mw.Config.Sources) == 0 {
		return nil
	}

	// Process TODOS os sources, não apenas o primeiro
	for _, source := range mw.Config.Sources {
		// ENVIRONMENT FIXO - lê valores de configura YAML
		switch source.Type {
		case "fixed":
			for field, value := range source.AddTo {
				if _, exists := input.Fields[field]; !exists {
					// Converta o valor de configuração YAML para structpb.Value
					input.Fields[field] = u.interfaceToStructPb(value)
				}
			}

		case "graphql":
		// 	// Implementar com u.api.CallAPI
		// 	// Para GraphQL, simula a resposta
		// 	if _, exists := input.Fields["limitMaximo"]; !exists {
		// 		// Aqui você pode usar valores do AddTo ou valores fixos de fallback
		// 		if fixedValue, hasFixed := source.AddTo["limiteMaximo"]; hasFixed {
		// 			input.Fields["limiteMaximo"] = u.interfaceToStructPb(fixedValue)
		// 		} else {
		// 			input.Fields["limiteMaximo"] = structpb.NewNumberValue(500.0)
		// 		}
		// 	}
		// }
		}
	}
	return nil
}

// interfaceToStructPb converte interface{} para structpb.Value baseado no tipo
func (u *ProcessRequestUseCase) interfaceToStructPb(value interface{}) *structpb.Value {
	switch v := value.(type) {
	case float64:
		return structpb.NewNumberValue(v)
	case int:
		return structpb.NewNumberValue(float64(v))
	case string:
		return structpb.NewStringValue(v)
	case bool:
		return structpb.NewBoolValue(v)
	case nil:
		return structpb.NewNullValue()
	default:
		// Tenta converter para float64
		if floatVal, ok := v.(float64); ok {
			return structpb.NewNumberValue(floatVal)
		}
		return structpb.NewStringValue(fmt.Sprintf("%v", v))
	}
}

func (u *ProcessRequestUseCase) processStep(step *domain.Step, input, output *structpb.Struct) error {
	switch step.Type {
	case "validate":
		ok, err := u.cel.EvalBool(step.Expr, input, output)
		if err != nil {
			log.Printf("ERROR: Validate CEL error: %v", err)
			// Retorna um erro que indica que é um erro de validação
			return &domain.ValidationError{Message: err.Error(), Code: 400}
		}
		if !ok {
			errorMsg := step.OnFail.Msg
			if errorMsg == ""{
				errorMsg = "Validação falhou"
			}
			// Retorna erro com código embutido
			return &domain.ValidationError{Message: errorMsg, Code: step.OnFail.Code}
		}

	case "transform":
		val, err := u.cel.EvalValue(step.Expr, input, output)
		if err != nil {
			return err
		}
		// Adiciona ao output
		if v, ok := val.(float64); ok {
			output.Fields["result"] = structpb.NewNumberValue(v)
		}

	case "output":
		// Processa o mapeamento de campos para filtrar o output
		return u.processOutputMapping(step, input, output)

	case "metrics":
		_ = u.datadog.Incr(step.Datadog.Metric, step.Datadog.Value, step.Datadog.Tags)

	case "log":
		u.log.Log(step.Msg, input)
	}

	return nil
}

// processOutputMapping filtra e mapeia campos conforme definido no YAML
func (u *processRequestUseCase) processOutputMapping(step *domain.Step, input, output *structpb.Struct) error {
	if step.Fields == nil {
		return nil
	}

	// Cria um novo struct apenas com os campos mapeados
	filteredOutput := &structpb.Struct{
		Fields: make(map[string]*structpb.Value),
	}

	// Processa CADA campo do mapeamento
	for fieldKey, fieldValue := range step.Fields {
		switch v := fieldValue.(type) {
		case string:
			// Campo simples - tenta extrair valor
			value := u.extractFieldValue(v, input, output)
			if value != nil {
				filteredOutput.Fields[fieldKey] = value
			}

		case map[interface{}]interface{}:
			// Campo nested (YAML usa interface{} como chave)
			nestedStruct := &structpb.Struct{
				Fields: make(map[string]*structpb.Value),
			}

			hasNestedFields := false
			for nestedKey, nestedValue := range v {
				if keyStr, ok := nestedKey.(string); ok {
					if nestedExpr, ok := nestedValue.(string); ok {
						value := u.extractFieldValue(nestedExpr, input, output)
						if value != nil {
							nestedStruct.Fields[keyStr] = value
							hasNestedFields = true
						}
					}
				}
			}

			// Só adiciona o campo nested se tiver pelo menos um campo interno
			if hasNestedFields {
				filteredOutput.Fields[fieldKey] = structpb.NewStructValue(nestedStruct)
			}

		case map[string]interface{}:
			// Campo nested (alternativa)
			nestedStruct := &structpb.Struct{
				Fields: make(map[string]*structpb.Value),
			}

			hasNestedFields := false
			for nestedKey, nestedValue := range v {
				if nestedExpr, ok := nestedValue.(string); ok {
					value := u.extractFieldValue(nestedExpr, input, output)
					if value != nil {
						nestedStruct.Fields[nestedKey] = value
						hasNestedFields = true
					}
				}
			}

			if hasNestedFields {
				filteredOutput.Fields[fieldKey] = structpb.NewStructValue(nestedStruct)
			}
		}
	}

	output.Fields = filteredOutput.Fields
	return nil
}

// extractFieldValue extrai valor de um campo, retorna nil se não existir
func (u *ProcessRequestUseCase) extractFieldVaue(expr string, input, output *structpb.Struct) *structpb.Value {
	// Para "output.beneficios.desconto" - busca direto da estrutura existente
	if expr == "output.beneficios.desconto" {
		if beneficios, exists := output.Fields["beneficios"]; exists {
			if beneficiosStruct, ok := beneficios.GetKind().(*structpb.Value_StructValue); ok {
				if desconto, exists := beneficiosStruct.StructValue.Fields["desconto"]; exists {
					return desconto
				}
			}
		}
		return nil
	}

	// Tenta avaliar como expressão CEL
	val, err := u.cel.EvalValue(expr, input, output)
	if err == nil {
		return u.celValueToStructPb(val)
	}
	return nil
}

// extractNestedValue extrai valor de estrutura nested, retorna nil se não existir
func (u *ProcessRequestUseCase) extractNestedVaue(s *structpb.Struct, path string) *structpb.Value {
	parts := strings.Split(path, ".")
	current := s

	for i, part := range parts {
		if current == nil || current.Fields == nil {
			return nil
		}

		val, exists := current.Fields[part]
		if !exists {
			return nil
		}

		if i == len(parts)-1 {
			return val
		}

		// Continua navegando na estrutura nested
		if nested, ok := val.GetKind().(*structpb.Value_StructValue); ok {
			current = nested.StructValue
		} else {
			return nil
		}
	}
	return nil
}

// celValueToStructPb convert valores CEL para structpb.Value
func (u *ProcessRequestUseCase) celValueToStrutPb(val interface{}) *structpb.Value {
	switch v := val.(type) {
	case float64:
		return structpb.NewNumberValue(v)
	case int64:
		return structpb.NewNumberValue(float64(v))
	case string:
		return structpb.NewStringValue(v)
	case bool:
		return structpb.NewBoolValue(v)
	case nil:
		return structpb.NewNullValue()
	default:
		return structpb.NewStringValue("")
	}
}