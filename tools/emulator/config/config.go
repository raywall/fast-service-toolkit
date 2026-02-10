package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// Config representa a estrutura do JSON de configuração (Lista de Servers)
type Config []ServerConfig

// Load carrega a configuração do arquivo padrão (emulator.json) ou via variável de ambiente.
// Retorna uma configuração vazia se o arquivo não existir, para não quebrar a inicialização.
func Load() Config {
	cfg := make(Config, 0)

	path := os.Getenv("EMULATOR_CONFIG_PATH")
	if path == "" {
		path = "emulator.json"
	}

	// Tenta carregar. Se falhar, loga aviso mas retorna vazio (safe default)
	if err := cfg.LoadFromFile(path); err != nil {
		log.Printf("Aviso: Não foi possível carregar %s: %v. Iniciando sem rotas mockadas.", path, err)
		return cfg
	}

	return cfg
}

func (cfg *Config) LoadFromFile(filepath string) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("erro ao ler arquivo: %v", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("erro ao parsear json: %v", err)
	}
	return nil
}
