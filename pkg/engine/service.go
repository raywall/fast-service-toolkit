package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/raywall/fast-service-toolkit/pkg/auth"
	"github.com/raywall/fast-service-toolkit/pkg/config"
	"github.com/raywall/fast-service-toolkit/pkg/enrichment"
	"github.com/raywall/fast-service-toolkit/pkg/graphql"
	"github.com/raywall/fast-service-toolkit/pkg/logger"
	"github.com/raywall/fast-service-toolkit/pkg/metrics"
	"github.com/raywall/fast-service-toolkit/pkg/observability"
	"github.com/raywall/fast-service-toolkit/pkg/proxy"
	"github.com/raywall/fast-service-toolkit/pkg/responder"
	"github.com/raywall/fast-service-toolkit/pkg/rules"
	"github.com/rs/zerolog"
)

var interpolationRegex = regexp.MustCompile(`\$\{([^}]+)\}`)

type ServiceEngine struct {
	mu              sync.RWMutex
	ConfigSource    string
	Config          *config.ServiceConfig
	Logger          zerolog.Logger
	Metrics         metrics.Provider
	MetricProcessor *metrics.Processor
	RuleManager     *rules.RuleManager
	Responder       *responder.ResponseBuilder
	GraphQLEngine   *graphql.GraphQLEngine
	AuthManagers    map[string]*auth.Manager
}

func NewServiceEngine(cfg *config.ServiceConfig, configSource string) (*ServiceEngine, error) {
	log := logger.Configure(cfg.Service.Logging)

	metricProvider, err := observability.SetupMetrics(cfg.Service.Metrics)
	if err != nil {
		return nil, fmt.Errorf("falha métricas: %w", err)
	}

	rm, err := rules.NewRuleManager()
	if err != nil {
		return nil, fmt.Errorf("falha fatal ao iniciar RuleManager: %w", err)
	}

	// CHECK: Só inicializa Responder se Steps existirem
	var respBuilder *responder.ResponseBuilder
	if cfg.Steps != nil {
		var err error
		respBuilder, err = responder.NewResponseBuilder(cfg.Steps.Output, rm)
		if err != nil {
			return nil, fmt.Errorf("falha responder: %w", err)
		}
	}

	metricProcessor := metrics.NewProcessor(cfg.Service.Metrics.Datadog.CustomDefinitions, metricProvider, rm)

	authManagers := make(map[string]*auth.Manager)
	for _, mw := range cfg.Middlewares {
		if mw.Type == "auth_provider" {
			var authCfg auth.AuthConfig
			if err := decodeConfig(mw.Config, &authCfg); err != nil {
				return nil, fmt.Errorf("erro config auth '%s': %w", mw.ID, err)
			}
			mgr := auth.NewOAuth2Manager(authCfg)
			log.Info().Str("middleware_id", mw.ID).Msg("Iniciando Auth Manager...")
			if err := mgr.Start(context.Background()); err != nil {
				return nil, fmt.Errorf("erro fatal iniciando auth '%s': %w", mw.ID, err)
			}
			authManagers[mw.ID] = mgr
		}
	}

	var gqlEngine *graphql.GraphQLEngine
	if cfg.GraphQL.Enabled {
		gqlEngine, err = graphql.NewGraphQLEngine(cfg.GraphQL, rm)
		if err != nil {
			return nil, fmt.Errorf("falha ao iniciar engine graphql: %w", err)
		}
	}

	return &ServiceEngine{
		ConfigSource:    configSource,
		Config:          cfg,
		Logger:          log,
		Metrics:         metricProvider,
		MetricProcessor: metricProcessor,
		RuleManager:     rm,
		Responder:       respBuilder,
		GraphQLEngine:   gqlEngine,
		AuthManagers:    authManagers,
	}, nil
}

func (se *ServiceEngine) Execute(ctx context.Context, payload []byte) (int, []byte, map[string]string, error) {
	// Proteção para não quebrar se Steps for nil
	if se.Config.Steps == nil {
		return 500, errorJSON("Configuration Error: Steps not defined"), nil, nil
	}

	// 1. Parse Input
	var inputMap map[string]interface{}
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &inputMap); err != nil {
			se.Logger.Error().Err(err).Msg("JSON payload inválido")
			return 400, errorJSON("Invalid JSON payload"), nil, nil
		}
	} else {
		inputMap = make(map[string]interface{})
	}

	inputHeaders := make(map[string]string)
	if h, ok := ctx.Value("request_headers").(map[string]string); ok {
		inputHeaders = h
	}

	execCtx := map[string]interface{}{
		"input":     inputMap,
		"header":    inputHeaders,
		"env":       getEnvVars(),
		"vars":      make(map[string]interface{}),
		"detection": make(map[string]interface{}),
	}

	// 3. Middlewares
	for _, mw := range se.Config.Middlewares {
		switch mw.Type {
		case "enrichment":
			if err := se.executeEnrichmentMiddleware(ctx, mw, execCtx); err != nil {
				se.Logger.Error().Err(err).Msg("Falha crítica no Enrichment")
				return 500, errorJSON("Enrichment failed"), nil, nil
			}
		case "rate_limit":
			// TODO
		case "auth_provider":
			if mgr, exists := se.AuthManagers[mw.ID]; exists {
				token, err := mgr.Get()
				if err != nil {
					se.Logger.Error().Err(err).Str("mw_id", mw.ID).Msg("Falha ao recuperar token")
					return 500, errorJSON("Auth dependency failed"), nil, nil
				}
				if outVar, ok := mw.Config["output_var"].(string); ok && outVar != "" {
					if _, ok := execCtx["auth"]; !ok {
						execCtx["auth"] = make(map[string]interface{})
					}
					authMap := execCtx["auth"].(map[string]interface{})
					if _, ok := authMap[mw.ID]; !ok {
						authMap[mw.ID] = make(map[string]interface{})
					}
					authMap[mw.ID].(map[string]interface{})[outVar] = token
				}
			}
		}
	}

	// 4. Input Validation
	for _, rule := range se.Config.Steps.Input.Validations {
		ok, err := se.RuleManager.EvaluateBool(rule.Expr, execCtx)
		if err != nil {
			se.Logger.Error().Err(err).Str("rule_id", rule.ID).Msg("Erro validação input")
			return 500, errorJSON("Internal logic error"), nil, nil
		}
		if !ok {
			return rule.OnFail.Code, errorJSON(rule.OnFail.Msg), nil, nil
		}
	}

	// 5. Processing
	for _, rule := range se.Config.Steps.Processing.Validations {
		ok, err := se.RuleManager.EvaluateBool(rule.Expr, execCtx)
		if err != nil {
			se.Logger.Error().Err(err).Str("rule_id", rule.ID).Msg("Erro validação processing")
			return 500, errorJSON("Internal logic error"), nil, nil
		}
		if !ok {
			return rule.OnFail.Code, errorJSON(rule.OnFail.Msg), nil, nil
		}
	}

	vars := execCtx["vars"].(map[string]interface{})
	for _, transform := range se.Config.Steps.Processing.Transformations {
		res, err := se.RuleManager.ExecuteTransformation(transform, execCtx)
		if err != nil {
			se.Logger.Error().Err(err).Str("transform", transform.Name).Msg("Erro transformação")
			return 500, errorJSON("Transformation error"), nil, nil
		}
		if res.Applied {
			key := strings.TrimPrefix(res.Target, "vars.")
			vars[key] = res.Value
		}
	}

	// 6. Output Validation
	for _, rule := range se.Config.Steps.Output.Validations {
		ok, err := se.RuleManager.EvaluateBool(rule.Expr, execCtx)
		if err != nil {
			se.Logger.Error().Err(err).Str("rule_id", rule.ID).Msg("Erro validação output")
			return 500, errorJSON("Internal output error"), nil, nil
		}
		if !ok {
			return rule.OnFail.Code, errorJSON(rule.OnFail.Msg), nil, nil
		}
	}

	// 7. Output Build
	statusCode, respBody, respHeaders, err := se.Responder.Build(execCtx)
	if err != nil {
		se.Logger.Error().Err(err).Msg("Erro output build")
		return 500, errorJSON(fmt.Sprintf("Output build error: %v", err)), nil, nil
	}

	// 8. Interceptor
	if se.Config.Steps.Output.Target.URL != "" {
		targetURL, err := se.interpolateString(se.Config.Steps.Output.Target.URL, execCtx)
		if err != nil {
			se.Logger.Error().Err(err).Msg("Erro interpolando Target URL")
			return 500, errorJSON("Invalid Target URL"), nil, nil
		}

		method := se.Config.Steps.Output.Target.Method
		if method == "" {
			method = "POST"
		}

		se.Logger.Info().Str("target", targetURL).Msg("Interceptor: encaminhando requisição")
		downstreamResp, err := proxy.ForwardRequest(ctx, method, targetURL, respBody, respHeaders, se.Config.Steps.Output.Target.Timeout)
		if err != nil {
			se.Logger.Error().Err(err).Str("target", targetURL).Msg("Falha na chamada downstream")
			return 502, errorJSON(fmt.Sprintf("Downstream error: %v", err)), nil, nil
		}

		statusCode = downstreamResp.StatusCode
		respBody = downstreamResp.Body
		respHeaders = downstreamResp.Headers
	}

	// 9. Métricas
	var respMap map[string]interface{}
	_ = json.Unmarshal(respBody, &respMap)
	execCtx["response"] = respMap

	if len(se.Config.Steps.Output.Metrics) > 0 {
		if err := se.MetricProcessor.ProcessRules(se.Config.Steps.Output.Metrics, execCtx); err != nil {
			se.Logger.Warn().Err(err).Msg("Falha ao registrar métricas de output")
		}
	}

	return statusCode, respBody, respHeaders, nil
}

// Reload ...
func (se *ServiceEngine) Reload() error {
	se.Logger.Info().Msgf("🔄 Hot Reload iniciado. Buscando config em: %s", se.ConfigSource)
	newCfg, err := Load(se.ConfigSource)
	if err != nil {
		return fmt.Errorf("falha ao carregar nova configuração: %w", err)
	}

	newRm, err := rules.NewRuleManager()
	if err != nil {
		return err
	}

	newMetricProcessor := metrics.NewProcessor(newCfg.Service.Metrics.Datadog.CustomDefinitions, se.Metrics, newRm)
	newAuthManagers := make(map[string]*auth.Manager)
	for _, mw := range newCfg.Middlewares {
		if mw.Type == "auth_provider" {
			var authCfg auth.AuthConfig
			if err := decodeConfig(mw.Config, &authCfg); err != nil {
				return fmt.Errorf("erro config auth reload '%s': %w", mw.ID, err)
			}
			mgr := auth.NewOAuth2Manager(authCfg)
			if err := mgr.Start(context.Background()); err != nil {
				return fmt.Errorf("erro iniciando auth reload '%s': %w", mw.ID, err)
			}
			newAuthManagers[mw.ID] = mgr
		}
	}

	var newGqlEngine *graphql.GraphQLEngine
	if newCfg.GraphQL.Enabled {
		newGqlEngine, err = graphql.NewGraphQLEngine(newCfg.GraphQL, newRm)
		if err != nil {
			return fmt.Errorf("falha ao recriar engine graphql: %w", err)
		}
	}

	se.mu.Lock()
	oldAuthManagers := se.AuthManagers
	se.Config = newCfg
	se.RuleManager = newRm
	se.GraphQLEngine = newGqlEngine
	se.AuthManagers = newAuthManagers
	se.MetricProcessor = newMetricProcessor

	if newCfg.Steps != nil {
		newRespBuilder, err := responder.NewResponseBuilder(newCfg.Steps.Output, newRm)
		if err == nil {
			se.Responder = newRespBuilder
		}
	}
	se.mu.Unlock()

	for _, mgr := range oldAuthManagers {
		mgr.Stop()
	}

	se.Logger.Info().Msg("✅ Hot Reload concluído com sucesso!")
	return nil
}

func (se *ServiceEngine) GetGraphQLEngine() *graphql.GraphQLEngine {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return se.GraphQLEngine
}

func (se *ServiceEngine) RunMiddlewares(ctx context.Context) (context.Context, error) {
	se.mu.RLock()
	defer se.mu.RUnlock()

	authContext := make(map[string]interface{})

	for _, mw := range se.Config.Middlewares {
		switch mw.Type {
		case "auth_provider":
			if mgr, exists := se.AuthManagers[mw.ID]; exists {
				token, err := mgr.Get()
				if err != nil {
					return nil, fmt.Errorf("auth '%s' not ready: %w", mw.ID, err)
				}
				if outVar, ok := mw.Config["output_var"].(string); ok && outVar != "" {
					if _, ok := authContext[mw.ID]; !ok {
						authContext[mw.ID] = make(map[string]interface{})
					}
					authContext[mw.ID].(map[string]interface{})[outVar] = token
				}
			}
		}
	}
	newCtx := context.WithValue(ctx, "auth_context", authContext)
	return newCtx, nil
}

func (se *ServiceEngine) executeEnrichmentMiddleware(ctx context.Context, mwConf config.MiddlewareConf, execCtx map[string]interface{}) error {
	var eConfig EnrichmentConfig
	if err := decodeConfig(mwConf.Config, &eConfig); err != nil {
		return fmt.Errorf("configuração enrichment inválida: %w", err)
	}

	detection := execCtx["detection"].(map[string]interface{})
	var wg sync.WaitGroup
	var mu sync.Mutex
	errChan := make(chan error, len(eConfig.Sources))

	for _, source := range eConfig.Sources {
		wg.Add(1)
		go func(src EnrichmentSource) {
			defer wg.Done()

			resolvedParams, err := se.resolveParams(src.Params, execCtx)
			if err != nil {
				errChan <- fmt.Errorf("erro resolvendo params source '%s': %w", src.Name, err)
				return
			}
			resolvedHeaders, err := se.resolveHeaders(src.Headers, execCtx)
			if err != nil {
				errChan <- fmt.Errorf("erro resolvendo headers source '%s': %w", src.Name, err)
				return
			}

			var result interface{}
			var callErr error

			switch src.Type {
			case "fixed":
				result, callErr = enrichment.ProcessFixed(resolvedParams)
			case "rest":
				method := toString(resolvedParams["method"])
				url := toString(resolvedParams["url"])
				body := resolvedParams["body"]
				result, callErr = enrichment.ProcessRest(ctx, method, url, resolvedHeaders, body)
			case "graphql":
				endpoint := toString(resolvedParams["endpoint"])
				query := toString(resolvedParams["query"])
				vars, _ := resolvedParams["variables"].(map[string]interface{})
				result, callErr = enrichment.ProcessGraphQL(ctx, endpoint, query, vars, resolvedHeaders)
			case "aws_parameter_store":
				region := toString(resolvedParams["region"])
				path := toString(resolvedParams["path"])
				decrypt := toBool(resolvedParams["with_decryption"])
				result, callErr = enrichment.ProcessAWSParameterStore(ctx, region, path, decrypt)
			case "aws_secrets_manager":
				region := toString(resolvedParams["region"])
				secretID := toString(resolvedParams["secret_id"])
				result, callErr = enrichment.ProcessAWSSecretsManager(ctx, region, secretID)
			case "aws_s3":
				region := toString(resolvedParams["region"])
				bucket := toString(resolvedParams["bucket"])
				key := toString(resolvedParams["key"])
				format := toString(resolvedParams["format"])
				result, callErr = enrichment.ProcessS3(ctx, region, bucket, key, format)
			case "aws_dynamodb":
				region := toString(resolvedParams["region"])
				table := toString(resolvedParams["table"])
				keyMap := toMap(resolvedParams["key"])
				if len(keyMap) == 0 {
					callErr = fmt.Errorf("key vazia")
				} else {
					result, callErr = enrichment.ProcessDynamoDB(ctx, region, table, keyMap)
				}
			default:
				callErr = fmt.Errorf("tipo de source desconhecido: %s", src.Type)
			}

			if callErr != nil {
				se.Logger.Warn().Err(callErr).Str("source", src.Name).Msg("Falha na fonte de dados")
				return
			}
			mu.Lock()
			detection[src.Name] = result
			mu.Unlock()
		}(source)
	}

	wg.Wait()
	close(errChan)
	if len(errChan) > 0 {
		return <-errChan
	}
	return nil
}

func (se *ServiceEngine) interpolateString(input string, ctx map[string]interface{}) (string, error) {
	if !strings.Contains(input, "${") {
		return input, nil
	}

	var replaceErr error
	result := interpolationRegex.ReplaceAllStringFunc(input, func(match string) string {
		expr := match[2 : len(match)-1]
		val, err := se.RuleManager.EvaluateValue(expr, ctx)
		if err != nil {
			replaceErr = fmt.Errorf("falha ao interpolar '%s': %w", match, err)
			return match
		}
		return toString(val)
	})

	if replaceErr != nil {
		return "", replaceErr
	}
	return result, nil
}

func (se *ServiceEngine) resolveParams(raw map[string]interface{}, ctx map[string]interface{}) (map[string]interface{}, error) {
	resolved := make(map[string]interface{})
	for k, v := range raw {
		if strVal, ok := v.(string); ok {
			val, err := se.interpolateString(strVal, ctx)
			if err != nil {
				return nil, err
			}
			resolved[k] = val
		} else {
			resolved[k] = v
		}
	}
	return resolved, nil
}

func (se *ServiceEngine) resolveHeaders(raw map[string]string, ctx map[string]interface{}) (map[string]string, error) {
	resolved := make(map[string]string)
	for k, v := range raw {
		val, err := se.interpolateString(v, ctx)
		if err != nil {
			return nil, err
		}
		resolved[k] = val
	}
	return resolved, nil
}

func decodeConfig(input interface{}, output interface{}) error {
	cleanInput := sanitizeMap(input)
	data, err := json.Marshal(cleanInput)
	if err != nil {
		return fmt.Errorf("configuração enrichment inválida: %w", err)
	}
	return json.Unmarshal(data, output)
}

func sanitizeMap(input interface{}) interface{} {
	switch x := input.(type) {
	case map[interface{}]interface{}:
		m := map[string]interface{}{}
		for k, v := range x {
			m[fmt.Sprintf("%v", k)] = sanitizeMap(v)
		}
		return m
	case map[string]interface{}:
		m := map[string]interface{}{}
		for k, v := range x {
			m[k] = sanitizeMap(v)
		}
		return m
	case []interface{}:
		l := make([]interface{}, len(x))
		for i, v := range x {
			l[i] = sanitizeMap(v)
		}
		return l
	default:
		return input
	}
}

func errorJSON(msg string) []byte {
	return []byte(fmt.Sprintf(`{"error": "%s"}`, msg))
}

func getEnvVars() map[string]string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) == 2 {
			env[pair[0]] = pair[1]
		}
	}
	return env
}

func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func toBool(v interface{}) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

func toMap(v interface{}) map[string]interface{} {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	if mGen, ok := v.(map[interface{}]interface{}); ok {
		out := make(map[string]interface{})
		for k, val := range mGen {
			out[fmt.Sprintf("%v", k)] = val
		}
		return out
	}
	return nil
}
