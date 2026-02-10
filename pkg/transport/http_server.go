package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/raywall/fast-service-lab/pkg/engine"
	"github.com/rs/zerolog/log"
)

const (
	HeaderCorrelationID = "x-correlation-id"
	HeaderLatency       = "x-latency-ms"
	ContextKeyCorrID    = "correlation_id"
)

// Regex para identificar parâmetros na rota (ex: {id})
var routeParamRegex = regexp.MustCompile(`\{([a-zA-Z0-9_]+)\}`)

func StartHTTPServer(svc *engine.ServiceEngine) error {
	mux := http.NewServeMux()

	if svc.Config.GraphQL.Enabled {
		svc.Logger.Info().Msgf("Registrando GraphQL em %s", svc.Config.GraphQL.Route)
		mux.HandleFunc(svc.Config.GraphQL.Route, createGraphQLHandler(svc))
	}

	if svc.Config.Service.Route != "" && svc.Config.Service.Route != svc.Config.GraphQL.Route {
		svc.Logger.Info().Msgf("Registrando Service REST em %s", svc.Config.Service.Route)
		mux.HandleFunc(svc.Config.Service.Route, createRESTHandler(svc))
	}

	handler := ObservabilityMiddleware(mux)

	addr := fmt.Sprintf(":%d", svc.Config.Service.Port)
	svc.Logger.Info().Msgf("Servidor HTTP ouvindo em %s", addr)

	return http.ListenAndServe(addr, handler)
}

func createGraphQLHandler(svc *engine.ServiceEngine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mwCtx, err := svc.RunMiddlewares(r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf("Middleware error: %v", err), 429)
			return
		}

		var p struct {
			Query     string                 `json:"query"`
			Variables map[string]interface{} `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "Invalid JSON Body", 400)
			return
		}

		result := svc.GetGraphQLEngine().Execute(mwCtx, p.Query, p.Variables)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

// createRESTHandler evoluído para suportar Path e Query Params
func createRESTHandler(svc *engine.ServiceEngine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Prepara Mapa de Input Unificado
		inputData := make(map[string]interface{})

		// A. Parse Body (se houver)
		bodyBytes, err := io.ReadAll(r.Body)
		if err == nil && len(bodyBytes) > 0 {
			// Tenta fazer unmarshal se for JSON, senão ignora (ou poderia tratar como string)
			_ = json.Unmarshal(bodyBytes, &inputData)
		}
		defer r.Body.Close()

		// B. Parse Query Params (?id=5&filter=abc)
		for k, v := range r.URL.Query() {
			if len(v) > 0 {
				inputData[k] = v[0] // Pega o primeiro valor
			}
		}

		// C. Parse Path Params (/customer/{id})
		// Identificamos as variáveis configuradas na rota (ex: {id}) e extraímos da URL atual
		matches := routeParamRegex.FindAllStringSubmatch(svc.Config.Service.Route, -1)
		for _, match := range matches {
			if len(match) > 1 {
				paramName := match[1]
				// Go 1.22+: r.PathValue extrai o valor do wildcard
				paramValue := r.PathValue(paramName)
				if paramValue != "" {
					inputData[paramName] = paramValue
				}
			}
		}

		// 2. Recria o Payload para o Engine
		// O Engine espera []byte. Marshalizamos o mapa unificado.
		finalPayload, _ := json.Marshal(inputData)

		// 3. Configura Timeout e Contexto
		timeoutDuration, _ := time.ParseDuration(svc.Config.Service.Timeout)
		if timeoutDuration == 0 {
			timeoutDuration = 30 * time.Second
		}
		ctx, cancel := context.WithTimeout(r.Context(), timeoutDuration)
		defer cancel()

		// 4. Injeta Headers de Entrada
		reqHeaders := make(map[string]string)
		for k, v := range r.Header {
			if len(v) > 0 {
				reqHeaders[k] = v[0]
			}
		}
		ctx = context.WithValue(ctx, "request_headers", reqHeaders)

		// 5. Executa Engine
		code, resp, headers, err := svc.Execute(ctx, finalPayload)

		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("Erro crítico na execução REST")
			http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		for k, v := range headers {
			w.Header().Set(k, v)
		}

		w.WriteHeader(code)
		w.Write(resp)
	}
}

// --- MIDDLEWARE DE OBSERVABILIDADE (Mantido igual) ---
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode  int
	startTime   time.Time
	wroteHeader bool
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.statusCode = code
	duration := time.Since(rw.startTime)
	rw.Header().Set(HeaderLatency, fmt.Sprintf("%d", duration.Milliseconds()))
	rw.ResponseWriter.WriteHeader(code)
	rw.wroteHeader = true
}

func (rw *responseWriterWrapper) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

func ObservabilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		corrID := r.Header.Get(HeaderCorrelationID)
		if corrID == "" {
			corrID = uuid.NewString()
		}
		w.Header().Set(HeaderCorrelationID, corrID)

		logger := log.With().Str("correlation_id", corrID).Logger()
		ctx := logger.WithContext(r.Context())
		ctx = context.WithValue(ctx, ContextKeyCorrID, corrID)

		wrapper := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			startTime:      start,
		}

		next.ServeHTTP(wrapper, r.WithContext(ctx))

		logger.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", wrapper.statusCode).
			Int64("latency_ms", time.Since(start).Milliseconds()).
			Msg("request completed")
	})
}
