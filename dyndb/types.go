// dyndb/types.go
package dyndb

import (
	"context"
	"errors"

	"encoding/base64"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// ErrNotFound – erro padrão quando o item não existe
var ErrNotFound = errors.New("dyndb: item not found")

// DynamoDBClient interface para abstrair o cliente DynamoDB
type DynamoDBClient interface {
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	BatchWriteItem(ctx context.Context, params *dynamodb.BatchWriteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error)
	BatchGetItem(ctx context.Context, params *dynamodb.BatchGetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchGetItemOutput, error)
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
}

// Store — interface principal (genérica)
type Store[T any] interface {
	Get(ctx context.Context, hashKey, sortKey any) (*T, error)
	Put(ctx context.Context, item T) error
	Delete(ctx context.Context, hashKey, sortKey any) error

	BatchWrite(ctx context.Context, puts []T, deletes [][2]any) error
	BatchGet(ctx context.Context, keys [][2]any) ([]T, error)

	// Query e Scan retornam QueryBuilder[T]
	Query() *QueryBuilder[T]
	Scan() *QueryBuilder[T]
}

// GlobalSecondaryIndex para GSIs
type GlobalSecondaryIndex struct {
	Name           string               `env:"DYNAMODB_GSI_NAME"`
	HashKey        string               `env:"DYNAMODB_GSI_HASH_KEY"`
	SortKey        string               `env:"DYNAMODB_GSI_SORT_KEY"`
	ProjectionType types.ProjectionType `env:"DYNAMODB_GSI_PROJECTION_TYPE"`
}

// TableConfig — configuração da tabela
type TableConfig[T any] struct {
	TableName    string `env:"DYNAMODB_TABLE_NAME"`
	HashKey      string `env:"DYNAMODB_HASH_KEY"`
	SortKey      string `env:"DYNAMODB_SORT_KEY"`      // opcional
	TTLAttribute string `env:"DYNAMODB_TTL_ATTRIBUTE"` // opcional
}

// QueryFilter — tipo simples e que funciona 100% com genéricos
type QueryFilter[T any] func(*QueryBuilder[T])

// QueryBuilder — o builder fluente
type QueryBuilder[T any] struct {
	store       *dynamoStore[T]
	keyCond     *expression.KeyConditionBuilder
	filterCond  *expression.ConditionBuilder
	projection  *expression.ProjectionBuilder
	indexName   *string
	limit       *int32
	lastKey     map[string]types.AttributeValue
	scanForward *bool
	isScan      bool
}

func (qb *QueryBuilder[T]) Unmarshal(items []map[string]types.AttributeValue, lastKey map[string]types.AttributeValue) ([]T, string, error) {
	result := make([]T, 0, len(items))
	for _, item := range items {
		var t T
		if err := attributevalue.UnmarshalMap(item, &t); err != nil {
			return nil, "", err
		}
		result = append(result, t)
	}
	token := ""
	if lastKey != nil {
		if b, err := json.Marshal(lastKey); err == nil {
			token = base64.StdEncoding.EncodeToString(b)
		}
	}
	return result, token, nil
}
