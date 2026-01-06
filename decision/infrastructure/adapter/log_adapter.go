package adapter

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"regexp"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"
)

// LogAdapter para logs
type LogAdapter struct {
	handler *slog.JSONHandler
	logger *slog.Logger
}

// NewLogAdapter cria um novo adapter
func NewLogAdapter() *LogAdapter {
	// Configura o logger com TracingHandler
	handler := slog.NewJSONHandler(os.Stdout, nil)
	logAdapter := &LogAdapter{
		handler: handler,
		logger: slog.New(handler),
	}
	slog.SetDefault(logAdapter.logger)
	return logAdapter
}

// Log registra com placeholders no formato {input.field}
func (l *LogAdapter) Log(msg string, input *structpb.Struct) {
	// Processa placeholders no formato {input.fieldName}
	processedMsg := processPlaceholders(msg, input)
	log.Print(processedMsg)
}

// processPlaceholders substitui {input.fieldName} pelos valores reais
func processPlaceholders(msg string, input *structpb.Struct) string {
	result := msg

	// Regex para encontrar padrões {input.fieldName}
	re := regexp.MustCompile(`\{input\.(\w+)\}`)
	matches := re.FindAllStringSubmatch(result, -1)

	for _, match := range matches {
		if len(match) == 2 {
			fieldName := match[1] // Nome do campo sem "input."
			placeholder := match[0] // Texto completo {input.fieldName}
			value := extractValue(input, fieldName)
			result = strings.ReplaceAll(result, placeholder, value)
		}
	}

	return result
}

// extractValue extrai um valor como string do structpb.Struct
func extractValue(input *structpb.Struct, field string) string {
	if input == nil || input.Fields == nil {
		return "<no value>"
	}

	val, exists := input.Fields[field]
	if !exists || val == nil {
		return "<no value>"
	}

	// Converte baseado no tipo do campo
	switch v := val.Kind.(type) {
	case *structpb.Value_NumberValue:
		// Formata números sem decimais desnecessários
		if v.NumberValue == float64(int64(v.NumberValue)) {
			return fmt.Sprintf("%.0f", v.NumberValue) // Ex: 150.0 -> "150"
		}
		return fmt.Sprintf("%v", v.NumberValue)
	case *structpb.Value_StringValue:
		return v.StringValue
	case *structpb.Value_BoolValue:
		return fmt.Sprintf("%v", v.BoolValue)
	case *structpb.Value_NullValue:
		return "<null>"
	case *structpb.Value_StructValue:
		return "<struct>"
	case *structpb.Value_ListValue:
		return "<list>"
	default:
		return "<unknown>"
	}
}