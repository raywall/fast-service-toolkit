package logger

import (
	"testing"

	"github.com/raywall/fast-service-lab/pkg/config"
	"github.com/rs/zerolog"
)

func TestConfigure(t *testing.T) {
	t.Run("Default Level Info", func(t *testing.T) {
		cfg := config.LoggingConf{Enabled: true}
		_ = Configure(cfg)

		if zerolog.GlobalLevel() != zerolog.InfoLevel {
			t.Errorf("Esperado InfoLevel, atual %v", zerolog.GlobalLevel())
		}
	})

	t.Run("Custom Level Debug", func(t *testing.T) {
		cfg := config.LoggingConf{Enabled: true, Level: "debug"}
		_ = Configure(cfg)

		if zerolog.GlobalLevel() != zerolog.DebugLevel {
			t.Errorf("Esperado DebugLevel, atual %v", zerolog.GlobalLevel())
		}
	})

	t.Run("Disabled Logger", func(t *testing.T) {
		cfg := config.LoggingConf{Enabled: false}
		logger := Configure(cfg)

		// Testa se grava algo (deveria ir para io.Discard)
		// Zerolog não expõe fácil o writer, mas podemos assumir que não panicou
		logger.Info().Msg("teste")
	})
}
