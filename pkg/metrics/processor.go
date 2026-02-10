package metrics

import (
	"fmt"
	"strconv"

	"github.com/raywall/fast-service-lab/pkg/config"
	"github.com/raywall/fast-service-lab/pkg/rules"
)

// Processor gerencia a avaliação e envio de métricas.
type Processor struct {
	definitions map[string]MetricDefinition
	provider    Provider
	ruleManager *rules.RuleManager
}

// NewProcessor cria um processador linkando IDs de configuração aos seus tipos reais.
func NewProcessor(conf []config.CustomMetricDefinition, provider Provider, rm *rules.RuleManager) *Processor {
	defs := make(map[string]MetricDefinition)
	for _, d := range conf {
		defs[d.ID] = MetricDefinition{
			Name: d.Name,
			Type: MetricType(d.Type),
		}
	}

	return &Processor{
		definitions: defs,
		provider:    provider,
		ruleManager: rm,
	}
}

// ProcessRules avalia e registra uma lista de regras de métricas.
func (p *Processor) ProcessRules(rules []config.MetricRegistrationRule, ctx map[string]interface{}) error {
	for _, rule := range rules {
		if err := p.processSingleRule(rule, ctx); err != nil {
			return err // Fail fast ou logar e continuar? Geralmente logar, mas aqui retornamos erro.
		}
	}
	return nil
}

func (p *Processor) processSingleRule(rule config.MetricRegistrationRule, ctx map[string]interface{}) error {
	// 1. Buscar definição da métrica (Nome e Tipo)
	def, exists := p.definitions[rule.MetricID]
	if !exists {
		return fmt.Errorf("métrica não definida: %s", rule.MetricID)
	}

	// 2. Avaliar o Valor (CEL)
	rawVal, err := p.ruleManager.EvaluateValue(rule.Value, ctx)
	if err != nil {
		return fmt.Errorf("erro ao avaliar valor da métrica %s: %w", rule.MetricID, err)
	}

	val, err := toFloat64(rawVal)
	if err != nil {
		return fmt.Errorf("valor da métrica %s inválido: %w", rule.MetricID, err)
	}

	// 3. Avaliar Tags (CEL)
	var finalTags []string
	for k, expr := range rule.Tags {
		tagVal, err := p.ruleManager.EvaluateValue(expr, ctx)
		if err != nil {
			return fmt.Errorf("erro ao avaliar tag %s da métrica %s: %w", k, rule.MetricID, err)
		}
		finalTags = append(finalTags, fmt.Sprintf("%s:%v", k, tagVal))
	}

	// 4. Enviar para o Provider
	switch def.Type {
	case TypeCount:
		return p.provider.Count(def.Name, val, finalTags)
	case TypeGauge:
		return p.provider.Gauge(def.Name, val, finalTags)
	case TypeHistogram:
		return p.provider.Histogram(def.Name, val, finalTags)
	default:
		return fmt.Errorf("tipo de métrica desconhecido: %s", def.Type)
	}
}

// Helper para converter retorno do CEL (int, int64, float64, string) para float64
func toFloat64(v interface{}) (float64, error) {
	switch i := v.(type) {
	case float64:
		return i, nil
	case float32:
		return float64(i), nil
	case int:
		return float64(i), nil
	case int64:
		return float64(i), nil
	case string:
		return strconv.ParseFloat(i, 64)
	default:
		return 0, fmt.Errorf("tipo numérico não suportado: %T", v)
	}
}
