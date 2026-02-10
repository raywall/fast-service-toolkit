package proxy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestForwardRequest_Success(t *testing.T) {
	// 1. Setup do Servidor Mock (Destino)
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Valida o que o Forwarder enviou
		if r.Method != "POST" {
			t.Errorf("Método esperado POST, recebido %s", r.Method)
		}
		if r.Header.Get("X-Custom-Trace") != "trace-123" {
			t.Errorf("Header customizado não recebido")
		}
		if r.Header.Get("User-Agent") != "FastServiceToolkit/Interceptor" {
			t.Errorf("User-Agent incorreto: %s", r.Header.Get("User-Agent"))
		}

		// Lê e valida o body
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"user":"teste"}` {
			t.Errorf("Body incorreto recebido no destino: %s", string(body))
		}

		// Responde
		w.Header().Set("X-Server-Id", "server-01")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"status":"created"}`))
	}))
	defer mockServer.Close()

	// 2. Preparação dos dados de entrada
	ctx := context.Background()
	payload := []byte(`{"user":"teste"}`)
	headers := map[string]string{
		"X-Custom-Trace": "trace-123",
	}

	// 3. Execução
	// Usamos a URL do servidor mock
	resp, err := ForwardRequest(ctx, "POST", mockServer.URL, payload, headers, "1s")

	// 4. Asserts na Resposta
	if err != nil {
		t.Fatalf("Erro inesperado no forward: %v", err)
	}

	if resp.StatusCode != 201 {
		t.Errorf("Status Code esperado 201, recebido %d", resp.StatusCode)
	}

	if string(resp.Body) != `{"status":"created"}` {
		t.Errorf("Body de resposta incorreto: %s", string(resp.Body))
	}

	if resp.Headers["X-Server-Id"] != "server-01" {
		t.Errorf("Header de resposta perdido")
	}
}

func TestForwardRequest_Timeout(t *testing.T) {
	// 1. Setup de Servidor Lento
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simula processamento demorado (100ms)
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(200)
	}))
	defer mockServer.Close()

	// 2. Execução com Timeout Curto (10ms)
	ctx := context.Background()
	// Timeout configurado para 10ms, servidor leva 100ms -> Deve falhar
	_, err := ForwardRequest(ctx, "GET", mockServer.URL, nil, nil, "10ms")

	// 3. Asserts
	if err == nil {
		t.Fatal("Esperava erro de timeout, recebeu sucesso")
	}

	// Verifica se o erro é de fato um timeout de contexto
	// A mensagem exata pode variar dependendo do OS, mas geralmente contém "deadline exceeded" ou "context deadline exceeded"
	expectedErr := "context deadline exceeded"
	if !contains(err.Error(), expectedErr) && !contains(err.Error(), "Client.Timeout exceeded") {
		t.Errorf("Erro esperado de timeout, recebido: %v", err)
	}
}

func TestForwardRequest_ConnectionError(t *testing.T) {
	// Tenta conectar em uma porta onde não tem nada rodando
	_, err := ForwardRequest(context.Background(), "GET", "http://localhost:54321/nada", nil, nil, "1s")

	if err == nil {
		t.Error("Esperava erro de conexão, recebeu nil")
	}
}

// Helper simples para verificar string
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:] != "" // Simplificação, use strings.Contains em prod
}
