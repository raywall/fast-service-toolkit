package main

import (
	"context"
	"os"
	"testing"

	"github.com/raywall/fast-service-lab/pkg/engine"
)

func TestRun_ServerBootstrap(t *testing.T) {
	// 1. Cria Configuração Válida Temporária
	yamlContent := `
version: "1.0"
service:
  name: "boot-test"
  runtime: "local"
  port: 9999
  route: "/api"
  timeout: "1s"
  on_timeout: {code: 504, msg: "timeout"}
  logging: {level: "error", format: "json"}
  metrics: {datadog: {enabled: false}}
steps:
  input: {}
  processing: {}
  output: {status_code: 200, body: {}}
`
	tmp, _ := os.CreateTemp("", "server_test_*.yaml")
	defer os.Remove(tmp.Name())
	tmp.WriteString(yamlContent)
	tmp.Close()

	// 2. Mock do Starter para não bloquear o teste
	serverStarterCalled := false
	originalStarter := serverStarter

	// Substitui a função real por um Mock
	serverStarter = func(svc *engine.ServiceEngine) error {
		serverStarterCalled = true
		if svc.Config.Service.Name != "boot-test" {
			t.Errorf("Configuração não carregada corretamente. Nome: %s", svc.Config.Service.Name)
		}
		return nil
	}
	defer func() { serverStarter = originalStarter }()

	// 3. Executa a função run isolada (passando o path manualmente)
	err := run(context.Background(), tmp.Name())

	// 4. Validações
	if err != nil {
		t.Fatalf("Erro na inicialização do run: %v", err)
	}
	if !serverStarterCalled {
		t.Error("O servidor HTTP não foi iniciado")
	}
}
