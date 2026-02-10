package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config representa a estrutura do JSON de configuração
type Config []ServerConfig

func (cfg *Config) LoadFromFile(filepath string) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("Erro ao ler config.json: %v", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("Erro ao parsear config.json: %v", err)
	}
	return nil
}
