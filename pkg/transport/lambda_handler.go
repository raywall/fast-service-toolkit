package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/google/uuid"
	"github.com/raywall/fast-service-lab/pkg/engine"
	"github.com/rs/zerolog/log"
)

// LambdaHandler adapta eventos do API Gateway para a ServiceEngine
type LambdaHandler struct {
	svc *engine.ServiceEngine
}

// NewLambdaHandler cria uma nova instância do adaptador
func NewLambdaHandler(svc *engine.ServiceEngine) *LambdaHandler {
	return &LambdaHandler{svc: svc}
}

// Handle processa a requisição Lambda
func (h *LambdaHandler) Handle(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// 1. Observabilidade (Réplica da lógica do Middleware HTTP)
	start := time.Now()

	// Tenta obter correlation ID dos headers (Case sensitive no mapa do lambda pode variar dependendo do proxy,
	// mas geralmente buscamos headers específicos)
	corrID := req.Headers[HeaderCorrelationID]
	if corrID == "" {
		// Tenta lowercase caso o API Gateway tenha normalizado
		corrID = req.Headers["x-correlation-id"]
	}
	if corrID == "" {
		corrID = uuid.NewString()
	}

	// Configura Logger Contextual
	logger := log.With().Str("correlation_id", corrID).Logger()
	ctx = logger.WithContext(ctx)
	// Adiciona ID para uso interno da engine se necessário
	ctx = context.WithValue(ctx, ContextKeyCorrID, corrID)

	// 2. Roteamento (GraphQL vs REST)
	var response events.APIGatewayProxyResponse
	var err error

	// Verifica se a rota batida corresponde à rota GraphQL configurada
	if h.svc.Config.GraphQL.Enabled && req.Path == h.svc.Config.GraphQL.Route {
		response, err = h.handleGraphQL(ctx, req)
	} else {
		response, err = h.handleREST(ctx, req)
	}

	// 3. Log Final (Similar ao middleware HTTP)
	duration := time.Since(start).Milliseconds()
	logger.Info().
		Str("method", req.HTTPMethod).
		Str("path", req.Path).
		Int("status", response.StatusCode).
		Int64("latency_ms", duration).
		Msg("lambda request completed")

	// Injeta headers de observabilidade na resposta
	if response.Headers == nil {
		response.Headers = make(map[string]string)
	}
	response.Headers[HeaderCorrelationID] = corrID

	return response, err
}

func (h *LambdaHandler) handleGraphQL(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// 1. Executa Middlewares de Negócio
	mwCtx, err := h.svc.RunMiddlewares(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 429,
			Body:       `{"error": "Middleware rejected request"}`,
		}, nil
	}

	// 2. Parse do Body
	var p struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables"`
	}
	if err := json.Unmarshal([]byte(req.Body), &p); err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       `{"error": "Invalid JSON Body"}`,
		}, nil
	}

	// 3. Executa GraphQL
	result := h.svc.GetGraphQLEngine().Execute(mwCtx, p.Query, p.Variables)

	responseBody, _ := json.Marshal(result)

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(responseBody),
	}, nil
}

func (h *LambdaHandler) handleREST(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Configura Timeout baseado na config (ou default lambda context, mas vamos respeitar a config do toolkit)
	timeoutDuration, _ := time.ParseDuration(h.svc.Config.Service.Timeout)
	if timeoutDuration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeoutDuration)
		defer cancel()
	}

	// 1. Injeta Headers de Entrada
	// APIGatewayProxyRequest já tem Headers map[string]string
	ctx = context.WithValue(ctx, "request_headers", req.Headers)

	// 2. Executa Engine (NOVA ASSINATURA)
	code, resp, headers, err := h.svc.Execute(ctx, []byte(req.Body))

	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Erro crítico na execução REST Lambda")
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       `{"error": "internal server error"}`,
		}, nil
	}

	// 3. Prepara Resposta com Headers
	respHeaders := map[string]string{"Content-Type": "application/json"}
	for k, v := range headers {
		respHeaders[k] = v
	}

	return events.APIGatewayProxyResponse{
		StatusCode: code,
		Headers:    respHeaders,
		Body:       string(resp),
	}, nil
}
