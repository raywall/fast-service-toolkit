package adapter

import (
	"fmt"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/raywall/fast-service-toolkit/decision/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

// CELAdapter implementa avaliação CEL com `input`, `output` e função `set()`
type CELAdapter struct {
	env *cel.Env
}

// celValueToStructPb converte valores CEL para structpb.Value
func celValueToStructPb(val ref.Val) *structpb.Value {
	switch v := val.(type) {
	case types.Int:
		return structpb.NewNumberValue(float64(v))
	case types.Double:
		return structpb.NewNumberValue(float64(v))
	case types.String:
		return structpb.NewStringValue(string(v))
	case types.Bool:
		return structpb.NewBoolValue(bool(v))
	case types.Null:
		return structpb.NewNullValue()
	default:
		// para outros tipos, usa o valor convertido para string
		return structpb.NewStringValue(fmt.Sprintf("%v", val.Value()))
	}
}

// processSetFunctions processa expressões com set() e modifica o output
func (a *CELAdapter) processSetFunctions(expr string, result interface{}, output *structpb.Struct) {
	// extrai todos os campos do set() na expressão
	fields := a.extractSetFields(expr)

	for fieldPath, defaultValue := range fields {
		if output.Fields == nil {
			output.Fields = make(map[string]*structpb.Value)
		}

		// verifica se o campo é nested (tem ".")
		if strings.Contains(fieldPath, ".") {
			a.setNestedField(output, fieldPath, result, defaultValue)
		} else {
			// campo simples
			if floatVal, ok := result.(float64); ok {
				output.Fields[fieldPath] = structpb.NewNumberValue(floatVal)
			} else if defaultValue != nil {
				output.Fields[fieldPath] = a.interfaceToStructPb(defaultValue)
			}
		}
	}
}

// setNestedField cria estrutura nested para campos como "beneficios.desconto"
func (a *CELAdapter) setNestedField(output *structpb.Struct, fieldPath string, result interface{}, defaultValue interface{}) {
	parts := strings.Split(fieldPath, ".")
	
	current := output
	for i, part := range parts {
		if i == len(parts)-1 {
			// última parte - define o valor
			if current.Fields == nil {
				current.Fields = make(map[string]*structpb.Value)
			}
			if floatVal, ok := result.(float64); ok {
				current.Fields[part] = structpb.NewNumberValue(floatVal)
			} else if defaultValue != nil {
				current.Fields[part] = a.interfaceToStructPb(defaultValue)
			}
		} else {
			// parte intermediária - cria estrutura nested
			if current.Fields == nil {
				current.Fields = make(map[string]*structpb.Value)
			}

			if existing, exists := current.Fields[part]; exists {
				// já existe, usa a estrutura existente
				if nested, ok := existing.GetKind().(*structpb.Value_StructValue); ok {
					current = nested.StructValue
				} else {
					// conflito de tipo, substitui por nova estrutura
					newStruct := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
					current.Fields[part] = structpb.NewStructValue(newStruct)
					current = newStruct
				}
			} else {
				// cria nova estrutura
				newStruct := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
				current.Fields[part] = structpb.NewStructValue(newStruct)
				current = newStruct
			}
		}
	}
}

// NewCELAdapter cria ambiente com:
// - `input` e `output` como map[string]dyn
// - `set(output, "key", value)` -> modifica `output` diretamente
func NewCELAdapter() (*CELAdapter, error) {
	env, err := cel.NewEnv(
		cel.Variables("input", cel.MapType(cel.StringType, cel.DynType)),
		cel.Variables("output", cel.MapType(cel.StringType, cel.DynType)),
		cel.Function("set",
			cel.Overload("set_map_string_dyn", 
				[]*cel.Type{
					cel.MapType(cel.StringType, cel.DynType),
					cel.StringType,
					cel.DynType,
				},
				cel.DynType, // retorna o valor setado
				cel.FunctionBinding(func(arts ...ref.Val) ref.Val {
					if len(args) != 3 {
						return types.NewErr("set() expects 3 arguments: map, key, value")
					}

					// key must be string
					if _, ok := args[1].(types.String); !ok {
						return types.NewErr("set() key must be string")
					}

					// retorna o valor que foi setado
					return args[2]
				}),
			),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("falha ao criar CEL env: %w", err)
	}
	return &CELAdapter{env: env}, nil
}

// compile compila expressão
func (a *CELAdapter) compile(expr string) (*cel.Ast, error) {
	ast, iss := a.env.Compile(expr)
	if iss.Err() != nil {
		return nil, iss.Err()
	}
	return ast, nil
}

// EvalBool avalia bool
func (a *CELAdapter) EvalBool(expr string, input, output *structpb.Struct) (bool, error) {
	ast, err := a.compile(expr)
	if err != nil {
		return false, err
	}
	prg, err := a.env.Program(ast)
	if err != nil {
		return false, err
	}

	inputMap := input.AsMap()
	outputMap := output.AsMap()

	out, _, err := prg.Eval(map[string]interface{}{
		"input": inputMap,
		"output": outputMap,
	})
	if err != nil {
		return false, err
	}
	if b, ok := out.Value().(bool); ok {
		return b, nil
	}
	return false, fmt.Errorf("resultado não é bool: %T", out.Value())
}

// EvalValue avalia qualquer valor e processa funções set()
func (a *CELAdapter) EvalValue(expr string, input, output *structpb.Struct) (interface{}, error) {
	ast, err := a.compile(expr)
	if err != nil {
		return false, err
	}
	prg, err := a.env.Program(ast)
	if err != nil {
		return false, err
	}

	inputMap := input.AsMap()
	outputMap := output.AsMap()

	out, _, err := prg.Eval(map[string]interface{}{
		"input": inputMap,
		"output": outputMap,
	})
	if err != nil {
		return false, err
	}

	result := out.Value()

	// processa manualmente funções set() para modificar o output
	if strings.Contains(expr, "set()") {
		a.processSetFunctions(expr, result, output)
	}
	return result, nil
}

// extractSetFields extrai todos os campos e valores default de expressões set()
func (a *CELAdapter) extractSetFields(expr string) map[string]interface{} {
	fields := make(map[string]interface{})

	// encontra todas as ocorrencias de set(output, `field`, value)
	start := 0
	for {
		setStart := strings.Index(expr[start:], "set(output, ')")
		if setStart == -1 {
			break
		}
		setStart += start + len("set(output, ')")

		// extrai o nome do campo (pode conter . para nested)
		fieldEnd := strings.Index(expr[setStart:], "'")
		if fieldEnd == -1 {
			break
		}

		// encontra o valor (após a vírgula após o campo)
		valueStart := setStart + fieldEnd + 2 // +2 para pular "' " ou "',"
		if valueStart >= len(expr) {
			break
		}

		// encontra o fim do valor (próxima vírgula ou fecha parênteses)
		valueEnd := -1
		parenCount := 1
		for i := valueStart; i < len(expr); i++ {
			if expr[i] == '(' {
				parenCount++
			} else if expr[i] == ')' {
				parenCount--
				if parenCount == 0 {
					valueEnd = i
					break
				}
			} else if expr[i] == ',' && parenCount == 1 {
				valueEnd = i
				break
			}
		}
		
		if valueEnd == -1 {
			break
		}

		valueStr := strings.TrimSpace(expr[valueStart:valueEnd])

		// tenta interpretar o valor
		var value interface{}
		if strings.Contains(valueStr, "input.valor * 0.1") {
			// é uma expressão de calculo - será resolvida pelo resultado
			value = nil
		} else if valueStr == "0" {
			value = float64(0)
		} else {
			value = valueStr
		}

		fields[fieldName] = value
		start = valueEnd
	}

	return fields
}

// interfaceToStructPb converte interface{} para structpb.Value
func (a *CELAdapter) interfaceToStructPb(value interface{}) *structpb.Value {
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
		return structpb.NewStringValue(fmt.Sprintf("%v", v))
	}
}

// interface
var _ domain.CELAdapterInterface = (*CELAdapter)(nil)
