package config

import (
	"os"
	"testing"
)

func TestConfig_LoadFromFile(t *testing.T) {
	// JSON Válido
	validJSON := `[
		{
			"port": 8080,
			"routes": [
				{
					"path": "/users",
					"method": "GET",
					"response": {"status": 200, "body": {"ok": true}}
				}
			]
		}
	]`

	tmp, _ := os.CreateTemp("", "valid_*.json")
	defer os.Remove(tmp.Name())
	tmp.WriteString(validJSON)
	tmp.Close()

	var cfg Config
	err := cfg.LoadFromFile(tmp.Name())

	if err != nil {
		t.Fatalf("Erro ao carregar: %v", err)
	}

	if len(cfg) != 1 {
		t.Fatalf("Deveria ter 1 server config")
	}
	if cfg[0].Port != 8080 {
		t.Errorf("Porta incorreta")
	}
	if cfg[0].Routes[0].Path != "/users" {
		t.Errorf("Rota incorreta")
	}
}

func TestConfig_Load_InvalidFile(t *testing.T) {
	var cfg Config
	err := cfg.LoadFromFile("arquivo_inexistente.json")
	if err == nil {
		t.Error("Deveria falhar com arquivo inexistente")
	}
}

func TestConfig_Load_BadJSON(t *testing.T) {
	tmp, _ := os.CreateTemp("", "bad_*.json")
	defer os.Remove(tmp.Name())
	tmp.WriteString(`{ "não é um array": true }`) // Config espera []ServerConfig
	tmp.Close()

	var cfg Config
	err := cfg.LoadFromFile(tmp.Name())
	if err == nil {
		t.Error("Deveria falhar com JSON malformado")
	}
}
