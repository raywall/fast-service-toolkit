package enrichment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HttpClientInterface permite mockar o cliente HTTP nos testes.
type HttpClientInterface interface {
	Do(req *http.Request) (*http.Response, error)
}

var Client HttpClientInterface = &http.Client{Timeout: 10 * time.Second} // Aumentei timeout para segurança

// ProcessFixed retorna dados estáticos definidos na configuração.
func ProcessFixed(params map[string]interface{}) (interface{}, error) {
	if val, ok := params["value"]; ok {
		return val, nil
	}
	// Fallback: Se não tiver a chave 'value', retorna o mapa inteiro
	return params, nil
}

// ProcessRest realiza uma chamada REST e retorna o dado processado (Map, Slice ou String).
func ProcessRest(ctx context.Context, method, url string, headers map[string]string, body interface{}) (interface{}, error) {
	// 1. Prepara o Body
	var bodyReader io.Reader
	if body != nil {
		if strBody, ok := body.(string); ok {
			bodyReader = strings.NewReader(strBody)
		} else {
			jsonBody, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("erro ao codificar body: %w", err)
			}
			bodyReader = bytes.NewBuffer(jsonBody)
		}
	}

	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(method), url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar request: %w", err)
	}

	// Headers Default
	req.Header.Set("Content-Type", "application/json")
	// Definir User-Agent para evitar bloqueio 403 (Cloudflare/WAF)
	req.Header.Set("User-Agent", "FastServiceToolkit/1.0 (Bot)")

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// 2. Executa
	resp, err := Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro na chamada REST: %w", err)
	}
	defer resp.Body.Close()

	// Lê todo o corpo para memória (permite tentar JSON ou String)
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro lendo resposta: %w", err)
	}

	// 3. Verifica Status de Erro HTTP
	if resp.StatusCode >= 400 {
		// Retorna erro formatado, mas inclui o corpo para debug se possível
		return nil, fmt.Errorf("http error %d: %s", resp.StatusCode, string(respBytes))
	}

	// 4. Parse Inteligente
	if len(respBytes) == 0 {
		return nil, nil
	}

	// ALTERAÇÃO: Usar Decoder com UseNumber para manter inteiros como inteiros
	// Isso ajuda o graphql-go a fazer o cast correto
	var result interface{}
	decoder := json.NewDecoder(bytes.NewReader(respBytes))
	decoder.UseNumber()

	if err := decoder.Decode(&result); err == nil {
		return result, nil
	}

	// Se não for JSON, retorna string
	return string(respBytes), nil
}

// ProcessGraphQL realiza uma query GraphQL e retorna 'data'.
func ProcessGraphQL(ctx context.Context, endpoint string, query string, variables map[string]interface{}, headers map[string]string) (interface{}, error) {
	payload := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}

	rawResult, err := ProcessRest(ctx, "POST", endpoint, headers, payload)
	if err != nil {
		return nil, err
	}

	resultMap, ok := rawResult.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("resposta GraphQL inválida (não é JSON object)")
	}

	if errorsVal, hasErrors := resultMap["errors"]; hasErrors && errorsVal != nil {
		return nil, fmt.Errorf("erros retornados pelo GraphQL: %v", errorsVal)
	}

	return resultMap["data"], nil
}
