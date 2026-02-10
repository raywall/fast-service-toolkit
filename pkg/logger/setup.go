package logger

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/raywall/fast-service-toolkit/pkg/config"
	"github.com/rs/zerolog"
)

// Configure inicializa o logger global baseando-se na configuração do YAML.
func Configure(cfg config.LoggingConf) zerolog.Logger {
	// Define o nível de log (default: info)
	level, err := zerolog.ParseLevel(strings.ToLower(cfg.Level))
	if err != nil || cfg.Level == "" {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Define o output (JSON para produção, Console "bonito" para local se solicitado)
	var output io.Writer = os.Stdout
	if !cfg.Enabled {
		output = io.Discard
	} else if cfg.Format == "console" {
		output = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	}

	// Cria o logger com contexto padrão
	logger := zerolog.New(output).
		With().
		Timestamp().
		Logger()

	return logger
}
