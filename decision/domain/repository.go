package domain

// ConfigRepository define interface para carregar configurações
type ConfigRepository interface {
	LoadConfig() (*Config, error)
	GetConfig() *Config
}