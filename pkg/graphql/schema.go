package graphql

import (
	"fmt"
	"strings"

	"github.com/graphql-go/graphql"
	"github.com/raywall/fast-service-lab/pkg/config"
	"github.com/raywall/fast-service-lab/pkg/rules"
)

// buildSchema constrói o objeto Schema do GraphQL
func buildSchema(conf config.GraphQLConf, rm *rules.RuleManager) (graphql.Schema, error) {
	typesMap := make(map[string]*graphql.Object)

	// 1. Cria Stubs (Objetos vazios)
	for name := range conf.Types {
		typesMap[name] = graphql.NewObject(graphql.ObjectConfig{Name: name, Fields: graphql.Fields{}})
	}

	// 2. Preenche Campos
	for name, typeDef := range conf.Types {
		obj := typesMap[name]
		fields, err := buildFields(typeDef.Fields, typesMap, rm)
		if err != nil {
			return graphql.Schema{}, fmt.Errorf("erro tipo %s: %w", name, err)
		}
		for fName, fConfig := range fields {
			obj.AddFieldConfig(fName, fConfig)
		}
	}

	// 3. Root Query
	queryFields, err := buildFields(conf.Query, typesMap, rm)
	if err != nil {
		return graphql.Schema{}, fmt.Errorf("erro query: %w", err)
	}

	schemaConfig := graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{Name: "Query", Fields: queryFields}),
	}

	return graphql.NewSchema(schemaConfig)
}

// buildFields itera sobre a config e cria os campos GraphQL com resolvers
func buildFields(fieldsConf map[string]config.GQLField, typesMap map[string]*graphql.Object, rm *rules.RuleManager) (graphql.Fields, error) {
	fields := graphql.Fields{}
	for name, fieldDef := range fieldsConf {
		gqlType, err := resolveType(fieldDef.Type, typesMap)
		if err != nil {
			return nil, fmt.Errorf("campo %s: %w", name, err)
		}

		argsConf := graphql.FieldConfigArgument{}
		for argName, argTypeStr := range fieldDef.Args {
			argType, err := resolveType(argTypeStr, typesMap)
			if err != nil {
				return nil, err
			}
			argsConf[argName] = &graphql.ArgumentConfig{Type: argType}
		}

		fieldConfig := &graphql.Field{
			Type: gqlType,
			Args: argsConf,
		}

		// CORREÇÃO CRÍTICA:
		// Só usamos o resolver customizado (assíncrono/complexo) se houver uma fonte de dados (Source).
		// Se Source for nil, usamos o comportamento padrão do GraphQL (ler propriedade do objeto pai).
		if fieldDef.Source != nil {
			fieldConfig.Resolve = makeResolver(fieldDef, rm)
		}

		fields[name] = fieldConfig
	}
	return fields, nil
}

func resolveType(typeName string, typesMap map[string]*graphql.Object) (graphql.Output, error) {
	isList := false
	if len(typeName) > 2 && typeName[0] == '[' && typeName[len(typeName)-1] == ']' {
		isList = true
		typeName = typeName[1 : len(typeName)-1]
	}

	var baseType graphql.Output

	// Normaliza tipos
	switch strings.ToLower(typeName) {
	case "string":
		baseType = graphql.String
	case "int":
		baseType = graphql.Int
	case "boolean":
		baseType = graphql.Boolean
	case "float":
		baseType = graphql.Float
	case "id":
		baseType = graphql.ID
	default:
		if obj, ok := typesMap[typeName]; ok {
			baseType = obj
		} else {
			return nil, fmt.Errorf("tipo desconhecido: %s", typeName)
		}
	}

	if isList {
		return graphql.NewList(baseType), nil
	}
	return baseType, nil
}
