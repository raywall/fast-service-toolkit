package main

// import (
// 	"os"
// 	"testing"

// 	"github.com/raywall/fast-service-toolkit/tools/emulator/config"
// )

// func TestEmulator_Main_Run(t *testing.T) {
// 	// 1. Cria Configuração JSON Temporária
// 	jsonContent := `[
// 		{
// 			"port": 9090,
// 			"routes": [{"path": "/test", "method": "GET", "response": {"status": 200}}]
// 		},
// 		{
// 			"port": 9091,
// 			"routes": [{"path": "/admin", "method": "POST", "response": {"status": 201}}]
// 		}
// 	]`
// 	tmp, _ := os.CreateTemp("", "emulator_config_*.json")
// 	defer os.Remove(tmp.Name())
// 	tmp.WriteString(jsonContent)
// 	tmp.Close()

// 	// 2. Mock do Starter
// 	// Conta quantos servidores tentaram subir
// 	startCount := 0
// 	originalStarter := serverStarter
// 	serverStarter = func(s *config.ServerConfig) {
// 		startCount++
// 	}
// 	defer func() { serverStarter = originalStarter }()

// 	// 3. Executa
// 	err := run(tmp.Name())

// 	// 4. Validações
// 	if err != nil {
// 		t.Fatalf("Erro ao rodar emulador: %v", err)
// 	}

// 	// O JSON define 2 servidores (portas 9090 e 9091)
// 	if startCount != 2 {
// 		t.Errorf("Esperado iniciar 2 servidores, iniciou %d", startCount)
// 	}
// }
