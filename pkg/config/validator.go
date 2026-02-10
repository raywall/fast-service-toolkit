package config

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

type ConfigValidator struct {
	validate *validator.Validate
}

// NewValidator cria uma nova instância do validador
func NewValidator() *ConfigValidator {
	return &ConfigValidator{
		validate: validator.New(),
	}
}

// Validate realiza validações estruturais (tags) e semânticas (lógica)
func (cv *ConfigValidator) Validate(cfg *ServiceConfig) error {
	// 1. Validação Estrutural (Tags do struct: required, oneof, etc)
	if err := cv.validate.Struct(cfg); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			var errMsgs []string
			for _, e := range validationErrors {
				errMsgs = append(errMsgs, fmt.Sprintf("Campo '%s' falhou na regra '%s'", e.Field(), e.Tag()))
			}
			return fmt.Errorf("erros de validação estrutural:\n- %s", strings.Join(errMsgs, "\n- "))
		}
		return fmt.Errorf("erro de validação estrutural: %w", err)
	}

	// 2. Validação Semântica (Regras de negócio da configuração)
	if err := cv.validateSemantics(cfg); err != nil {
		return fmt.Errorf("erro de validação semântica: %w", err)
	}

	return nil
}

func (cv *ConfigValidator) validateSemantics(cfg *ServiceConfig) error {
	// 1. Validação de Unicidade de IDs de Middleware
	seenIDs := make(map[string]bool)
	for _, mw := range cfg.Middlewares {
		if seenIDs[mw.ID] {
			return fmt.Errorf("middleware ID duplicado detectado: '%s'", mw.ID)
		}
		seenIDs[mw.ID] = true
	}

	// 2. Validação Condicional de Steps
	if cfg.Steps == nil {
		// Se não tem Steps, DEVE ser um serviço GraphQL habilitado
		if !cfg.GraphQL.Enabled && cfg.Service.Type != "graphql" {
			return fmt.Errorf("serviço inválido: 'steps' são obrigatórios a menos que 'graphql.enabled' seja true")
		}
		// Se for GraphQL e steps for nil, está tudo bem, retornamos aqui para evitar o crash
		return nil
	}

	// --- A partir daqui, cfg.Steps é garantido como não-nulo ---

	// 3. Validação do Interceptor Mode (Target)
	if cfg.Steps.Output.Target.URL != "" {
		method := strings.ToUpper(cfg.Steps.Output.Target.Method)
		validMethods := map[string]bool{"POST": true, "PUT": true, "PATCH": true, "GET": true, "DELETE": true, "": true} // "" default POST
		if !validMethods[method] {
			return fmt.Errorf("método HTTP inválido no target: '%s'. Use POST, PUT, GET, PATCH ou DELETE", method)
		}
	}

	// 4. Validação Cruzada: Transformações referenciando variáveis inexistentes
	// (Implementação futura mais complexa poderia analisar AST do CEL)

	return nil
}
