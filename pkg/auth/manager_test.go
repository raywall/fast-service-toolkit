package auth

import (
	"context"
	"testing"
	"time"
)

// MockFetcher simula a obtenção de tokens
func mockFetcher(token string, ttl time.Duration, err error) TokenFetcher {
	return func(ctx context.Context) (string, time.Duration, error) {
		return token, ttl, err
	}
}

func TestManager_Get(t *testing.T) {
	t.Run("Deve retornar token válido do cache", func(t *testing.T) {
		mgr := NewManager(mockFetcher("valid-token", 1*time.Hour, nil))
		// Inicializa manualmente para simular estado
		mgr.token = "valid-token"
		mgr.initialized = true

		token, err := mgr.Get()
		if err != nil {
			t.Fatalf("Erro inesperado: %v", err)
		}
		if token != "valid-token" {
			t.Errorf("Token incorreto: %s", token)
		}
	})

	t.Run("Deve retornar erro se não inicializado", func(t *testing.T) {
		mgr := NewManager(mockFetcher("", 0, nil))
		_, err := mgr.Get()
		if err == nil {
			t.Error("Deveria falhar se Start() não foi chamado")
		}
	})
}

func TestManager_Lifecycle(t *testing.T) {
	// Simula um token que expira muito rápido (10ms)
	// O Manager subtrai uma margem de segurança, então precisamos configurar para que a renovação ocorra.
	// Nota: O Manager real usa um timer. Testar tempo exato em unit test é flaky.
	// Vamos testar se o Start popula o token inicial.

	fetchCount := 0
	fetcher := func(ctx context.Context) (string, time.Duration, error) {
		fetchCount++
		return "token-1", 1 * time.Hour, nil
	}

	mgr := NewManager(fetcher)

	// Start deve buscar o primeiro token síncronamente
	err := mgr.Start(context.Background())
	if err != nil {
		t.Fatalf("Start falhou: %v", err)
	}

	if fetchCount != 1 {
		t.Errorf("Start deveria ter chamado fetcher 1 vez, chamou %d", fetchCount)
	}

	if mgr.token != "token-1" {
		t.Errorf("Token inicial não armazenado")
	}

	mgr.Stop()
}
