package domain

import "time"

// Service representa as configurações do serviço
type Service struct {
	Name string `yaml:"name"`
	Runtime string `yaml:"runtime"`
	Logging LoggingConfig `yaml:"logging,omitempty"`
	Metrics MetricsConfig `yaml:"metrics,omitempty"`
	Timeout time.Duration `yaml:"timeout"`
	OnTimeout OnFail `yaml:"on_timeout,omitempty"`
	Route string `yaml:"route"`
	Port int `yaml:"port"`
}

// LoggingConfig configura os logs
type LoggingConfig struct {
	Enabled bool `yaml:"enabled"`
	Level string `yaml:"level,omitempty"`
}

// MetricsConfig configura métricas
type MetricsConfig struct {
	Enabled bool `yaml:"enabled"`
	Datadog DatadogConfig `yaml:"datadog,omitempty"`
}

// DatadogConfig para integração com Datadog
type DatadogConfig struct {
	Addr string `yaml:"addr,omitempty"`
}

// OnFail define ação em falha
type OnFail struct {
	Code int `yaml:"code"`
	Msg string `yaml:"msg"`
}