package observability

import (
	"testing"

	"github.com/raywall/fast-service-toolkit/pkg/config"
)

func TestSetupMetrics(t *testing.T) {
	t.Run("Disabled returns Noop", func(t *testing.T) {
		cfg := config.MetricsConf{
			Datadog: config.DatadogConf{Enabled: false},
		}

		provider, err := SetupMetrics(cfg)
		if err != nil {
			t.Fatalf("Erro setup: %v", err)
		}

		if _, ok := provider.(*NoopProvider); !ok {
			t.Errorf("Esperado NoopProvider, recebido %T", provider)
		}
	})

	t.Run("Enabled returns Datadog", func(t *testing.T) {
		cfg := config.MetricsConf{
			Datadog: config.DatadogConf{
				Enabled: true,
				Addr:    "localhost:8125",
			},
		}

		provider, err := SetupMetrics(cfg)
		if err != nil {
			// statsd.New pode falhar se o endereço for inválido, mas localhost costuma passar na criação do struct
			t.Fatalf("Erro setup: %v", err)
		}

		if _, ok := provider.(*DatadogProvider); !ok {
			t.Errorf("Esperado DatadogProvider, recebido %T", provider)
		}
	})
}
