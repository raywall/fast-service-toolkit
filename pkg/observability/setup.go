package observability

import (
	"fmt"

	"github.com/DataDog/datadog-go/v5/statsd"
	"github.com/raywall/fast-service-lab/pkg/config"
	"github.com/raywall/fast-service-lab/pkg/metrics"
)

// NoopProvider é um placeholder para quando métricas estão desabilitadas.
type NoopProvider struct{}

func (n *NoopProvider) Count(name string, value float64, tags []string) error     { return nil }
func (n *NoopProvider) Gauge(name string, value float64, tags []string) error     { return nil }
func (n *NoopProvider) Histogram(name string, value float64, tags []string) error { return nil }

// DatadogProvider adapta a lib oficial do Datadog para nossa interface.
type DatadogProvider struct {
	client *statsd.Client
}

func (d *DatadogProvider) Count(name string, value float64, tags []string) error {
	return d.client.Count(name, int64(value), tags, 1)
}

func (d *DatadogProvider) Gauge(name string, value float64, tags []string) error {
	return d.client.Gauge(name, value, tags, 1)
}

func (d *DatadogProvider) Histogram(name string, value float64, tags []string) error {
	return d.client.Histogram(name, value, tags, 1)
}

// SetupMetrics inicializa o provedor correto baseado no YAML.
func SetupMetrics(cfg config.MetricsConf) (metrics.Provider, error) {
	if !cfg.Datadog.Enabled {
		return &NoopProvider{}, nil
	}

	// Configurações do cliente StatsD
	opts := []statsd.Option{
		statsd.WithNamespace(cfg.Datadog.Namespace),
	}

	client, err := statsd.New(cfg.Datadog.Addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar no datadog statsd: %w", err)
	}

	return &DatadogProvider{client: client}, nil
}
