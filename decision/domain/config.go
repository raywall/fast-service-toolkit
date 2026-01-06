package domain

// Config representa a configuração global do serviço
type Config struct {
	Service    Service      `yaml:"service"`
	Steps      Steps        `yaml:"steps"`
	Middleware []Middleware `yaml:"middleware"`
}
