package metrics

// Provider define o contrato para envio de métricas.
// Isso permite trocar Datadog por Prometheus ou Logging sem alterar a lógica de negócio.
type Provider interface {
	Count(name string, value float64, tags []string) error
	Gauge(name string, value float64, tags []string) error
	Histogram(name string, value float64, tags []string) error
}

// MetricType define os tipos suportados.
type MetricType string

const (
	TypeCount     MetricType = "count"
	TypeGauge     MetricType = "gauge"
	TypeHistogram MetricType = "histogram"
)

// MetricDefinition armazena os metadados da métrica (nome real, tipo).
type MetricDefinition struct {
	Name string
	Type MetricType
}
