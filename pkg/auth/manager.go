package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ========================================================================
// 1. CONFIGURAÇÃO E ESTRUTURAS AUXILIARES
// ========================================================================

type AuthConfig struct {
	TokenURL     string `yaml:"token_url" json:"token_url"`
	ClientID     string `yaml:"client_id" json:"client_id"`
	ClientSecret string `yaml:"client_secret" json:"client_secret"`
	Scope        string `yaml:"scope" json:"scope"`
}

// tokenResponse mapeia a resposta padrão da RFC 6749 (OAuth2)
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"` // Tempo em segundos
	TokenType   string `json:"token_type"`
}

// ========================================================================
// 2. CORE MANAGER (Sua implementação original, mantida intacta)
// ========================================================================

// TokenFetcher define a função que sabe como buscar um novo token.
type TokenFetcher func(ctx context.Context) (string, time.Duration, error)

// Manager gerencia o ciclo de vida do token de forma thread-safe.
type Manager struct {
	token       string
	mu          sync.RWMutex
	fetcher     TokenFetcher
	stopChan    chan struct{}
	initialized bool
}

// NewManager cria um gerenciador genérico.
func NewManager(fetcher TokenFetcher) *Manager {
	return &Manager{
		fetcher:  fetcher,
		stopChan: make(chan struct{}),
	}
}

// Start inicia o loop de renovação em background.
func (m *Manager) Start(ctx context.Context) error {
	// 1. Busca inicial síncrona
	token, ttl, err := m.fetcher(ctx)
	if err != nil {
		return fmt.Errorf("falha inicial ao obter token: %w", err)
	}

	m.setToken(token)
	m.initialized = true

	// 2. Inicia Goroutine de renovação
	go m.refreshLoop(ctx, ttl)

	return nil
}

// Get retorna o token atual de forma segura.
func (m *Manager) Get() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.initialized {
		return "", fmt.Errorf("auth manager não inicializado")
	}
	return m.token, nil
}

// Stop encerra o processo de renovação.
func (m *Manager) Stop() {
	close(m.stopChan)
}

func (m *Manager) setToken(t string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.token = t
}

func (m *Manager) refreshLoop(ctx context.Context, initialTTL time.Duration) {
	waitDuration := calculateWait(initialTTL)
	timer := time.NewTimer(waitDuration)

	for {
		select {
		case <-m.stopChan:
			timer.Stop()
			return
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			token, newTTL, err := m.fetcher(ctx)
			if err == nil {
				m.setToken(token)
				waitDuration = calculateWait(newTTL)
			} else {
				// Log de erro deve ser feito externamente ou injetando logger no Manager
				// Fallback curto em caso de erro
				waitDuration = 10 * time.Second
			}
			timer.Reset(waitDuration)
		}
	}
}

func calculateWait(ttl time.Duration) time.Duration {
	// Renova quando passar 80% do tempo de vida (margem de segurança)
	if ttl == 0 {
		return 5 * time.Minute // Fallback se a API não retornar expires_in
	}
	return time.Duration(float64(ttl) * 0.8)
}

// ========================================================================
// 3. IMPLEMENTAÇÃO OAUTH2 (CLIENT CREDENTIALS)
// ========================================================================

// NewOAuth2Manager é um helper que cria o Manager já configurado para Client Credentials.
func NewOAuth2Manager(cfg AuthConfig) *Manager {
	fetcher := NewOAuth2Fetcher(cfg)
	return NewManager(fetcher)
}

// NewOAuth2Fetcher cria a função de busca específica para o fluxo Client Credentials.
func NewOAuth2Fetcher(cfg AuthConfig) TokenFetcher {
	return func(ctx context.Context) (string, time.Duration, error) {
		// 1. Prepara os dados do Form (application/x-www-form-urlencoded)
		data := url.Values{}
		data.Set("grant_type", "client_credentials") // <--- OBRIGATÓRIO
		data.Set("client_id", cfg.ClientID)
		data.Set("client_secret", cfg.ClientSecret)

		if cfg.Scope != "" {
			data.Set("scope", cfg.Scope)
		}

		// 2. Cria a Requisição
		req, err := http.NewRequestWithContext(ctx, "POST", cfg.TokenURL, strings.NewReader(data.Encode()))
		if err != nil {
			return "", 0, fmt.Errorf("erro ao criar request: %w", err)
		}

		// Headers essenciais
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/json")

		// 3. Executa
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return "", 0, fmt.Errorf("erro de conexão oauth: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			return "", 0, fmt.Errorf("oauth provider retornou erro: %d", resp.StatusCode)
		}

		// 4. Parse da Resposta
		var tokenResp tokenResponse
		if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
			return "", 0, fmt.Errorf("erro decode json token: %w", err)
		}

		if tokenResp.AccessToken == "" {
			return "", 0, fmt.Errorf("access_token veio vazio")
		}

		// Converte expires_in (int seconds) para Duration
		ttl := time.Duration(tokenResp.ExpiresIn) * time.Second

		return tokenResp.AccessToken, ttl, nil
	}
}
