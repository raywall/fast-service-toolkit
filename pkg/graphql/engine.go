package graphql

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/graphql-go/graphql"
	"github.com/raywall/fast-service-toolkit/pkg/config"
	"github.com/raywall/fast-service-toolkit/pkg/enrichment"
	"github.com/raywall/fast-service-toolkit/pkg/rules"
)

var interpolationRegex = regexp.MustCompile(`\$\{([^}]+)\}`)

type GraphQLEngine struct {
	Schema      graphql.Schema
	RuleManager *rules.RuleManager
}

func NewGraphQLEngine(cfg config.GraphQLConf, rm *rules.RuleManager) (*GraphQLEngine, error) {
	engine := &GraphQLEngine{
		RuleManager: rm,
	}

	schema, err := engine.buildSchema(cfg)
	if err != nil {
		return nil, err
	}

	engine.Schema = schema
	return engine, nil
}

func (ge *GraphQLEngine) Execute(ctx context.Context, query string, variables map[string]interface{}) *graphql.Result {
	params := graphql.Params{
		Schema:         ge.Schema,
		RequestString:  query,
		VariableValues: variables,
		Context:        ctx,
	}
	return graphql.Do(params)
}

func (ge *GraphQLEngine) buildSchema(cfg config.GraphQLConf) (graphql.Schema, error) {
	objects := make(map[string]*graphql.Object)

	// 1. Declara objetos
	for name, typeDef := range cfg.Types {
		objects[name] = graphql.NewObject(graphql.ObjectConfig{
			Name:        name,
			Description: typeDef.Description,
			Fields:      graphql.Fields{},
		})
	}

	// 2. Preenche campos
	for name, typeDef := range cfg.Types {
		obj := objects[name]
		fields := graphql.Fields{}
		for fieldName, fieldDef := range typeDef.Fields {
			fields[fieldName] = ge.buildField(fieldDef, objects)
		}
		for k, v := range fields {
			obj.AddFieldConfig(k, v)
		}
	}

	// 3. Root Query
	rootQueryFields := graphql.Fields{}
	for name, fieldDef := range cfg.Query {
		rootQueryFields[name] = ge.buildField(fieldDef, objects)
	}

	// 4. Root Mutation
	var rootMutation *graphql.Object
	if len(cfg.Mutation) > 0 {
		rootMutationFields := graphql.Fields{}
		for name, fieldDef := range cfg.Mutation {
			rootMutationFields[name] = ge.buildField(fieldDef, objects)
		}
		rootMutation = graphql.NewObject(graphql.ObjectConfig{
			Name:   "Mutation",
			Fields: rootMutationFields,
		})
	}

	return graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name:   "Query",
			Fields: rootQueryFields,
		}),
		Mutation: rootMutation,
	})
}

func (ge *GraphQLEngine) buildField(def config.GQLField, objects map[string]*graphql.Object) *graphql.Field {
	gqlType := ge.resolveType(def.Type, objects)

	args := graphql.FieldConfigArgument{}
	for argName, argTypeStr := range def.Args {
		args[argName] = &graphql.ArgumentConfig{
			Type: ge.resolveType(argTypeStr, objects),
		}
	}

	return &graphql.Field{
		Type:        gqlType,
		Description: def.Description,
		Args:        args,
		Resolve:     ge.createResolver(def.Source),
	}
}

func (ge *GraphQLEngine) resolveType(typeStr string, objects map[string]*graphql.Object) graphql.Output {
	typeStr = strings.TrimSpace(typeStr)

	if strings.HasPrefix(typeStr, "[") && strings.HasSuffix(typeStr, "]") {
		innerType := typeStr[1 : len(typeStr)-1]
		return graphql.NewList(ge.resolveType(innerType, objects))
	}

	if strings.HasSuffix(typeStr, "!") {
		innerType := typeStr[:len(typeStr)-1]
		return graphql.NewNonNull(ge.resolveType(innerType, objects))
	}

	switch typeStr {
	case "String":
		return graphql.String
	case "Int":
		return graphql.Int
	case "Float":
		return graphql.Float
	case "Boolean":
		return graphql.Boolean
	case "ID":
		return graphql.ID
	default:
		if obj, ok := objects[typeStr]; ok {
			return obj
		}
		return graphql.String
	}
}

func (ge *GraphQLEngine) createResolver(src *config.EnrichmentSourceConfig) graphql.FieldResolveFn {
	if src == nil {
		return graphql.DefaultResolveFn
	}

	return func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context
		evalCtx := map[string]interface{}{
			"args":   p.Args,
			"source": p.Source,
			"env":    getEnvVars(),
		}

		if authCtx, ok := ctx.Value("auth_context").(map[string]interface{}); ok {
			evalCtx["auth"] = authCtx
		}

		resolvedParams, err := ge.resolveMap(src.Params, evalCtx)
		if err != nil {
			return nil, err
		}
		resolvedHeaders, err := ge.resolveHeaders(src.Headers, evalCtx)
		if err != nil {
			return nil, err
		}

		// Variáveis para capturar o resultado
		var result interface{}
		var resErr error

		switch src.Type {
		case "fixed":
			result, resErr = enrichment.ProcessFixed(resolvedParams)

		case "rest":
			method := toString(resolvedParams["method"])
			url := toString(resolvedParams["url"])
			body := resolvedParams["body"]
			result, resErr = enrichment.ProcessRest(ctx, method, url, resolvedHeaders, body)

		case "graphql":
			endpoint := toString(resolvedParams["endpoint"])
			query := toString(resolvedParams["query"])
			vars := toMap(resolvedParams["variables"])
			result, resErr = enrichment.ProcessGraphQL(ctx, endpoint, query, vars, resolvedHeaders)

		case "aws_dynamodb":
			region := toString(resolvedParams["region"])
			table := toString(resolvedParams["table"])
			key := toMap(resolvedParams["key"])
			result, resErr = enrichment.ProcessDynamoDB(ctx, region, table, key)

		default:
			return nil, fmt.Errorf("adapter desconhecido: %s", src.Type)
		}

		if resErr != nil {
			return nil, resErr
		}

		// Lógica de Extração (Response Path)
		// Se o usuário configurou "response_path", navegamos no resultado para retornar apenas o sub-objeto
		if path, ok := resolvedParams["response_path"].(string); ok && path != "" {
			if resultMap, ok := result.(map[string]interface{}); ok {
				if val, found := resultMap[path]; found {
					return val, nil
				}
				// Se o caminho não existe, retorna null ou o próprio objeto (comportamento resiliente)
				// Aqui optamos por retornar null para indicar "não encontrado"
				return nil, nil
			}
		}

		return result, nil
	}
}

// --- Helpers de Resolução Recursiva (Mantidos) ---

func (ge *GraphQLEngine) resolveMap(raw map[string]interface{}, ctx map[string]interface{}) (map[string]interface{}, error) {
	resolved := make(map[string]interface{})
	for k, v := range raw {
		res, err := ge.resolveRecursive(v, ctx)
		if err != nil {
			return nil, err
		}
		resolved[k] = res
	}
	return resolved, nil
}

func (ge *GraphQLEngine) resolveRecursive(input interface{}, ctx map[string]interface{}) (interface{}, error) {
	switch v := input.(type) {
	case string:
		return ge.interpolate(v, ctx)

	case map[string]interface{}:
		res := make(map[string]interface{})
		for k, val := range v {
			r, err := ge.resolveRecursive(val, ctx)
			if err != nil {
				return nil, err
			}
			res[k] = r
		}
		return res, nil

	case map[interface{}]interface{}:
		res := make(map[string]interface{})
		for k, val := range v {
			strKey := fmt.Sprintf("%v", k)
			r, err := ge.resolveRecursive(val, ctx)
			if err != nil {
				return nil, err
			}
			res[strKey] = r
		}
		return res, nil

	case []interface{}:
		res := make([]interface{}, len(v))
		for i, val := range v {
			r, err := ge.resolveRecursive(val, ctx)
			if err != nil {
				return nil, err
			}
			res[i] = r
		}
		return res, nil

	default:
		return v, nil
	}
}

func (ge *GraphQLEngine) resolveHeaders(raw map[string]string, ctx map[string]interface{}) (map[string]string, error) {
	resolved := make(map[string]string)
	for k, v := range raw {
		val, err := ge.interpolate(v, ctx)
		if err != nil {
			return nil, err
		}
		resolved[k] = toString(val)
	}
	return resolved, nil
}

func (ge *GraphQLEngine) interpolate(input string, ctx map[string]interface{}) (interface{}, error) {
	if strings.Contains(input, "${") {
		var replaceErr error
		result := interpolationRegex.ReplaceAllStringFunc(input, func(match string) string {
			expr := match[2 : len(match)-1]
			val, err := ge.RuleManager.EvaluateValue(expr, ctx)
			if err != nil {
				replaceErr = fmt.Errorf("err interpolating %s: %w", match, err)
				return match
			}
			return toString(val)
		})

		if replaceErr != nil {
			return nil, replaceErr
		}
		return result, nil
	}

	if strings.ContainsAny(input, "().+-*/><=") {
		val, err := ge.RuleManager.EvaluateValue(input, ctx)
		if err == nil {
			return val, nil
		}
	}
	return input, nil
}
