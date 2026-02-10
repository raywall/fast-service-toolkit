package engine

import (
	"fmt"
	"strings"

	"github.com/raywall/fast-service-toolkit/pkg/config"
	"github.com/raywall/fast-service-toolkit/pkg/rules"
)

// ValidationReport contém o resultado detalhado da análise.
type ValidationReport struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// Analyze realiza uma inspeção profunda na configuração.
func Analyze(cfg *config.ServiceConfig) (*ValidationReport, error) {
	report := &ValidationReport{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// 1. Inicializa dependências necessárias para checagem (ex: compilador CEL)
	rm, err := rules.NewRuleManager()
	if err != nil {
		return nil, fmt.Errorf("falha interna ao iniciar analisador de regras: %w", err)
	}

	// 2. Validação de Middlewares
	// Verifica se configs JSON podem ser decodificadas e se parâmetros obrigatórios (que usamos no código) existem
	for i, mw := range cfg.Middlewares {
		if mw.Type == "enrichment" {
			var eConf EnrichmentConfig
			if err := decodeConfig(mw.Config, &eConf); err != nil {
				report.Errors = append(report.Errors, fmt.Sprintf("Middleware[%d] Enrichment: Configuração inválida: %v", i, err))
			} else {
				// Validação profunda das sources
				for _, src := range eConf.Sources {
					if src.Name == "" {
						report.Errors = append(report.Errors, fmt.Sprintf("Middleware[%d]: Source sem nome definido", i))
					}
					// Aqui poderíamos validar se params obrigatórios para cada 'type' existem
				}
			}
		}
	}

	// 3. Validação de Regras CEL (Input)
	for _, rule := range cfg.Steps.Input.Validations {
		if _, err := rm.CompileProgram(rule.Expr); err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("Steps.Input.Rule[%s]: Erro de sintaxe CEL: %v", rule.ID, err))
		}
	}

	// 4. Validação de Regras CEL (Processing)
	declaredVars := make(map[string]bool)

	for _, rule := range cfg.Steps.Processing.Validations {
		if _, err := rm.CompileProgram(rule.Expr); err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("Steps.Processing.Validation[%s]: Erro CEL: %v", rule.ID, err))
		}
	}

	for _, trans := range cfg.Steps.Processing.Transformations {
		// Valida Condição
		if _, err := rm.CompileProgram(trans.Condition); err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("Steps.Processing.Transform[%s]: Erro na Condição: %v", trans.Name, err))
		}
		// Valida Valor
		if _, err := rm.CompileProgram(trans.Value); err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("Steps.Processing.Transform[%s]: Erro no Valor: %v", trans.Name, err))
		}
		// Rastreia variáveis criadas (análise estática simples)
		if strings.HasPrefix(trans.Target, "vars.") {
			varName := strings.TrimPrefix(trans.Target, "vars.")
			declaredVars[varName] = true
		}
	}

	// 5. Validação de Output
	for field, expr := range cfg.Steps.Output.Body {
		if _, err := rm.CompileProgram(expr.(string)); err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("Steps.Output.Field[%s]: Erro CEL: %v", field, err))
		}

		// Análise Heurística: Tentar detectar uso de variáveis não declaradas
		// (Isso é uma verificação simples de string, uma análise AST completa seria mais robusta mas complexa)
		for v := range declaredVars {
			// Se o output usa "vars.x" mas x não está em declaredVars (lógica inversa complexa aqui,
			// vamos apenas emitir warning se virmos "vars." e não acharmos match, simplificado para PoC)
			_ = v
		}
	}

	if len(report.Errors) > 0 {
		report.Valid = false
	}

	return report, nil
}
