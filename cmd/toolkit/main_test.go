package main

import (
	"os"
	"testing"
)

// TestRunValidate_HappyPath tenta validar um arquivo real.
func TestRunValidate_HappyPath(t *testing.T) {
	// 1. Cria config válida e completa para passar na validação estrita
	content := `
version: "1.0"
service:
  name: "cli-test"
  runtime: "local"
  port: 8080        # Obrigatório quando runtime é local
  route: "/cli"
  timeout: "1s"
  on_timeout: {code: 504, msg: "timeout"}
  logging:          # Obrigatório devido às validações 'oneof' sem omitempty
    level: "info"
    format: "console"
steps:
  input: {}
  processing: {}
  output: {status_code: 200, body: {}}
`
	tmp, _ := os.CreateTemp("", "cli_test_*.yaml")
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(content); err != nil {
		t.Fatalf("Erro escrevendo arquivo temp: %v", err)
	}
	tmp.Close()

	// 2. Executa a função (Redirecionando stdout se quisesse silenciar)
	// Se runValidate chamar os.Exit(1), o teste falha panicando.
	// Se passar liso, sucesso.
	runValidate(tmp.Name())
}
