package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewOAuth2Fetcher(t *testing.T) {
	// 1. Mock do Server OAuth2 (ex: Keycloak/Auth0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validações do protocolo OAuth2 Client Credentials
		if r.Method != "POST" {
			t.Errorf("Método esperado POST, recebido %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type incorreto")
		}

		err := r.ParseForm()
		if err != nil {
			t.Fatal(err)
		}

		if r.Form.Get("grant_type") != "client_credentials" {
			t.Errorf("grant_type incorreto: %s", r.Form.Get("grant_type"))
		}
		if r.Form.Get("client_id") != "my-client" {
			t.Errorf("client_id incorreto")
		}

		// Resposta de Sucesso
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token": "mock-jwt-xyz", "expires_in": 3600, "token_type": "Bearer"}`))
	}))
	defer server.Close()

	// 2. Configura o Fetcher
	cfg := AuthConfig{
		TokenURL:     server.URL,
		ClientID:     "my-client",
		ClientSecret: "my-secret",
	}

	fetcher := NewOAuth2Fetcher(cfg)

	// 3. Executa
	token, ttl, err := fetcher(context.Background())

	// 4. Valida
	if err != nil {
		t.Fatalf("Erro no fetcher: %v", err)
	}
	if token != "mock-jwt-xyz" {
		t.Errorf("Token retornado incorreto: %s", token)
	}
	if ttl.Seconds() != 3600 {
		t.Errorf("TTL incorreto: %v", ttl)
	}
}
