package rules

import (
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
)

// RuleManager gerencia a compilação e avaliação de expressões CEL.
type RuleManager struct {
	env *cel.Env
}

// NewRuleManager inicializa o ambiente CEL com as variáveis padrão esperadas.
func NewRuleManager() (*RuleManager, error) {
	// Cria o ambiente CEL com suporte a tipos dinâmicos (Dyn)
	env, err := cel.NewEnv(
		cel.StdLib(),
		cel.Declarations(
			decls.NewVar("input", decls.Dyn),     // O JSON de entrada
			decls.NewVar("vars", decls.Dyn),      // Variáveis temporárias
			decls.NewVar("env", decls.Dyn),       // Variáveis de ambiente
			decls.NewVar("detection", decls.Dyn), // Resultado de middlewares
			decls.NewVar("args", decls.Dyn),      // Argumentos GraphQL
			decls.NewVar("source", decls.Dyn),    // Source GraphQL
			decls.NewVar("auth", decls.Dyn),      // Dados de Autenticação
			decls.NewVar("header", decls.Dyn),    // Dados de Header
		),
	)
	if err != nil {
		return nil, fmt.Errorf("erro fatal CEL init: %w", err)
	}

	return &RuleManager{env: env}, nil
}

// EvaluateBool processa regras de validação (deve retornar true/false).
func (rm *RuleManager) EvaluateBool(expression string, ctx map[string]interface{}) (bool, error) {
	if expression == "" {
		return true, nil // Expressão vazia = aprova
	}

	ast, issues := rm.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return false, fmt.Errorf("erro compilação CEL '%s': %s", expression, issues.Err())
	}

	prg, err := rm.env.Program(ast)
	if err != nil {
		return false, fmt.Errorf("erro programa CEL: %w", err)
	}

	out, _, err := prg.Eval(ctx)
	if err != nil {
		return false, fmt.Errorf("erro execução CEL: %w", err)
	}

	// Converte resultado para bool
	if val, ok := out.Value().(bool); ok {
		return val, nil
	}
	return false, fmt.Errorf("resultado não é booleano")
}

// EvaluateValue processa regras de transformação (retorna um valor dinâmico).
func (rm *RuleManager) EvaluateValue(expression string, ctx map[string]interface{}) (interface{}, error) {
	if expression == "" {
		return nil, nil
	}

	ast, issues := rm.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("erro compilação CEL '%s': %s", expression, issues.Err())
	}

	prg, err := rm.env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("erro programa CEL: %w", err)
	}

	out, _, err := prg.Eval(ctx)
	if err != nil {
		return nil, fmt.Errorf("erro execução CEL: %w", err)
	}

	return out.Value(), nil
}

// CompileProgram expõe a compilação do CEL.
func (rm *RuleManager) CompileProgram(expr string) (cel.Program, error) {
	ast, issues := rm.env.Compile(expr)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("erro de compilação CEL: %w", issues.Err())
	}
	prg, err := rm.env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("erro ao gerar programa CEL: %w", err)
	}
	return prg, nil
}

// compile é um helper interno para compilar a string em um programa executável.
func (rm *RuleManager) compile(expr string) (cel.Program, error) {
	ast, issues := rm.env.Compile(expr)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("erro de compilação CEL: %w", issues.Err())
	}

	prg, err := rm.env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("erro ao gerar programa CEL: %w", err)
	}
	return prg, nil
}
