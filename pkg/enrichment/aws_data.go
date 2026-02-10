package enrichment

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// DynamoClient define a interface para operações do DynamoDB (permite Mock)
type DynamoClient interface {
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
}

// Cache de Clients (mantido da versão original)
var dynamoClient *dynamodb.Client

func getDynamoClient(ctx context.Context, region string) (*dynamodb.Client, error) {
	if dynamoClient != nil {
		return dynamoClient, nil
	}
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}
	dynamoClient = dynamodb.NewFromConfig(cfg)
	return dynamoClient, nil
}

// ProcessDynamoDB é o wrapper público que inicializa o cliente real.
func ProcessDynamoDB(ctx context.Context, region, table string, keyMap map[string]interface{}) (interface{}, error) {
	client, err := getDynamoClient(ctx, region)
	if err != nil {
		return nil, fmt.Errorf("erro config aws: %w", err)
	}
	// Delega para a função testável
	return processDynamoDBInternal(ctx, client, table, keyMap)
}

// processDynamoDBInternal contém a lógica pura, testável via Mock.
func processDynamoDBInternal(ctx context.Context, client DynamoClient, table string, keyMap map[string]interface{}) (interface{}, error) {
	// 1. Converter Map Go -> Map DynamoDB
	dbKey := make(map[string]types.AttributeValue)
	for k, v := range keyMap {
		if s, ok := v.(string); ok {
			dbKey[k] = &types.AttributeValueMemberS{Value: s}
		} else {
			// Conversão segura para string para chaves compostas
			dbKey[k] = &types.AttributeValueMemberS{Value: fmt.Sprintf("%v", v)}
		}
	}

	// 2. Executar GetItem
	input := &dynamodb.GetItemInput{
		TableName: aws.String(table),
		Key:       dbKey,
	}

	out, err := client.GetItem(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("operation error DynamoDB: GetItem, %w", err)
	}

	if out.Item == nil {
		return nil, nil // Not Found
	}

	// 3. Converter Resultado
	return unmarshalAttributeMap(out.Item), nil
}

// unmarshalAttributeMap converte map[string]AttributeValue para map[string]interface{}
func unmarshalAttributeMap(item map[string]types.AttributeValue) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range item {
		result[k] = unmarshalAttribute(v)
	}
	return result
}

func unmarshalAttribute(v types.AttributeValue) interface{} {
	switch val := v.(type) {
	case *types.AttributeValueMemberS:
		return val.Value
	case *types.AttributeValueMemberN:
		return val.Value
	case *types.AttributeValueMemberBOOL:
		return val.Value
	case *types.AttributeValueMemberM:
		return unmarshalAttributeMap(val.Value)
	case *types.AttributeValueMemberL:
		list := make([]interface{}, len(val.Value))
		for i, item := range val.Value {
			list[i] = unmarshalAttribute(item)
		}
		return list
	case *types.AttributeValueMemberNULL:
		return nil
	default:
		return nil
	}
}
