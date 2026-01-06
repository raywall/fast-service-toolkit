// Package adapter fornece adapters para serviços externos como APIs, CEL, etc
package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/machinebox/graphql"
	"github.com/raywall/fast-service-toolkit/decision/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

// APIAdapter é responsável por chamadas externas (GraphQL e REST)
type APIAdapter struct {
	cel domain.CELAdapterInterface // Para avaliar expressões em variables/query_params
}

// NewAPIAdapter cria uma nova instância do adapter
func NewAPIAdapter(cel domain.CELAdapterInterface) *APIAdapter {
	return &APIAdapter{cel: cel}
}

// CallAPI executa uma chamada externa com base na configuração da source
func (a *APIAdapter) CallAPI(ctx context.Context, source *domain.Source, input *structpb.Struct) (map[string]interface{}, error) {
	switch source.Type {
	case "graphql":
		return a.cellGraphQL(ctx, source, input)
	case "rest":
		return a.callREST(ctx, source, input)
	default:
		return nil, fmt.Errorf("api_type não suportado: %s", source.Type)
	}
}

// callGraphQL executa uma query GraphQL com variáveis dinâmicas
func (a *APIAdapter) callGraphQL(ctx context.Context, source *domain.Source, input *structpb.Struct) (map[string]interface{}, error) {
	client := graphql.NewClient(source.Endpoint)

	req := graphql.NewRequest(source.Query)

	// adiciona headers
	for key, value := range source.Headers {
		// Renderiza placeholders {{ .Env.VAR }}
		rendered := os.ExpandEnv(value)
		req.Header.Set(key, rendered)
	}

	// adiciona variáveis dinâmicas via CEL
	if source.Variables != nil {
		for varName, expr := range source.Variables {
			val, err := a.cel.EvalValue(expr, input, &structpb.Struct{})
			if err != nil {
				return nil, fmt.Errorf("falha ao avaliar variável %s: %w", varName, err)
			}
			req.Var(varName, val)
		}
	}

	// estrutura resposta
	var resp map[string]interface{}
	if err := client.Run(ctx, req, &resp); err != nil {
		return nil, fmt.Errorf("falha na chamada GraphQL: %w", err)
	}

	return resp, nil
}

// callREST executa uma requisição REST com suporte a query params e body dinâmico
func (a *APIAdapter) callREST(ctx context.Context, source *domain.Source, input *structpb.Struct) (map[string]interface{}, error) {
	method := http.MethodGet
	if source.Method != "" {
		method = source.Method
	}

	// monta URL com query params
	url := source.Endpoint
	if len(source.Variables) > 0 {
		// reutiliza Variables como query_params
		qp := make([]string, 0, len(source.Variables))
		for key, expr := range source.Variables {
			val, err := a.cel.EvalValue(expr, input, &structpb.Struct{})
			if err != nil {
				return nil, fmt.Errorf("falha ao avaliar query param %s: %w", key, err)
			}
			qp = append(qp, fmt.Sprintf("%s=%v", key, val))
		}
		if len(qp) > 0 {
			url = fmt.Sprintf("%s?%s", url, strings.Join(qp, "&"))
		}
	}

	var body io.Reader
	if source.Query != "" { 
		// reutiliza Query como body_expr
		val, err := a.cel.EvalValue(source.Query, input, &structpb.Struct{})
		if err != nil {
			return nil, fmt.Errorf("falha ao gerar body: %w", err)
		}
		jsonBody, _ := json.Marshal(val)
		body = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	// headers
	for key, value := range source.Headers {
		req.Header.Set(key, os.ExpandEnv(value))
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: falha na chamada REST", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}