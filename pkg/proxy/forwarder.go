package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Response representa a resposta do serviço downstream.
type Response struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
}

// Client reutilizável para pooling de conexões
var client = &http.Client{
	Timeout: 30 * time.Second, // Timeout padrão seguro
}

// ForwardRequest envia a requisição enriquecida para o serviço de destino.
func ForwardRequest(ctx context.Context, method, url string, body []byte, headers map[string]string, timeoutStr string) (*Response, error) {
	// 1. Configura Timeout específico se fornecido
	reqCtx := ctx
	if timeoutStr != "" {
		if dur, err := time.ParseDuration(timeoutStr); err == nil {
			var cancel context.CancelFunc
			reqCtx, cancel = context.WithTimeout(ctx, dur)
			defer cancel()
		}
	}

	// 2. Prepara Request
	req, err := http.NewRequestWithContext(reqCtx, strings.ToUpper(method), url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("erro ao criar forward request: %w", err)
	}

	// 3. Injeta Headers
	// User-Agent ajuda a identificar que a chamada veio do Interceptor
	req.Header.Set("User-Agent", "FastServiceToolkit/Interceptor")
	req.Header.Set("Content-Type", "application/json") // Default seguro, mas sobrescrito abaixo se existir no map

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// 4. Executa
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("falha na conexão com target (%s): %w", url, err)
	}
	defer resp.Body.Close()

	// 5. Lê Resposta
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler resposta do target: %w", err)
	}

	// 6. Extrai Headers de Resposta
	respHeaders := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			respHeaders[k] = v[0]
		}
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Headers:    respHeaders,
		Body:       respBody,
	}, nil
}
