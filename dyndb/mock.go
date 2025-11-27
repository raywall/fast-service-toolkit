// dyndb/mock_store.go
package dyndb

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// MockStore é um mock completo e fácil de usar para testes da interface Store[T].
//
// Ele expõe campos de função (`GetFn`, `PutFn`, etc.) que podem ser definidos
// para simular o comportamento desejado do DynamoDB durante os testes.
type MockStore[T any] struct {
	GetFn        func(ctx context.Context, hashKey, sortKey any) (*T, error)
	PutFn        func(ctx context.Context, item T) error
	DeleteFn     func(ctx context.Context, hashKey, sortKey any) error
	BatchWriteFn func(ctx context.Context, puts []T, deletes [][2]any) error
	BatchGetFn   func(ctx context.Context, keys [][2]any) ([]T, error)
	QueryFn      func() *MockQueryBuilder[T]
	ScanFn       func() *MockQueryBuilder[T]
}

func (m *MockStore[T]) Get(ctx context.Context, hashKey, sortKey any) (*T, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, hashKey, sortKey)
	}
	return nil, ErrNotFound
}

func (m *MockStore[T]) Put(ctx context.Context, item T) error {
	if m.PutFn != nil {
		return m.PutFn(ctx, item)
	}
	return nil
}

func (m *MockStore[T]) Delete(ctx context.Context, hashKey, sortKey any) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, hashKey, sortKey)
	}
	return nil
}

func (m *MockStore[T]) BatchWrite(ctx context.Context, puts []T, deletes [][2]any) error {
	if m.BatchWriteFn != nil {
		return m.BatchWriteFn(ctx, puts, deletes)
	}
	return nil
}

func (m *MockStore[T]) BatchGet(ctx context.Context, keys [][2]any) ([]T, error) {
	if m.BatchGetFn != nil {
		return m.BatchGetFn(ctx, keys)
	}
	return nil, nil
}

func (m *MockStore[T]) Query() *QueryBuilder[T] {
	if m.QueryFn != nil {
		return m.QueryFn().Builder
	}
	return (&MockQueryBuilder[T]{}).Builder
}

func (m *MockStore[T]) Scan() *QueryBuilder[T] {
	if m.ScanFn != nil {
		return m.ScanFn().Builder
	}
	return (&MockQueryBuilder[T]{}).Builder
}

// MockDynamoClient é um mock para a interface DynamoDBClient de baixo nível.
//
// Permite testar a lógica interna do `dynamoStore` sem tocar no AWS SDK.
type MockDynamoClient struct {
	GetItemFn        func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	PutItemFn        func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	DeleteItemFn     func(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	BatchWriteItemFn func(ctx context.Context, params *dynamodb.BatchWriteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error)
	BatchGetItemFn   func(ctx context.Context, params *dynamodb.BatchGetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchGetItemOutput, error)
}

func (m *MockDynamoClient) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	if m.GetItemFn != nil {
		return m.GetItemFn(ctx, params, optFns...)
	}
	return nil, ErrNotFound
}

func (m *MockDynamoClient) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	if m.GetItemFn != nil {
		return m.PutItemFn(ctx, params, optFns...)
	}
	return nil, ErrNotFound
}

func (m *MockDynamoClient) DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	if m.GetItemFn != nil {
		return m.DeleteItemFn(ctx, params, optFns...)
	}
	return nil, ErrNotFound
}

func (m *MockDynamoClient) BatchWriteItem(ctx context.Context, params *dynamodb.BatchWriteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error) {
	if m.GetItemFn != nil {
		return m.BatchWriteItemFn(ctx, params, optFns...)
	}
	return nil, ErrNotFound
}

func (m *MockDynamoClient) BatchGetItem(ctx context.Context, params *dynamodb.BatchGetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchGetItemOutput, error) {
	if m.GetItemFn != nil {
		return m.BatchGetItemFn(ctx, params, optFns...)
	}
	return nil, ErrNotFound
}

// MockQueryBuilder representa o mock do QueryBuilder fluente.
type MockQueryBuilder[T any] struct {
	Builder *QueryBuilder[T]
	ExecFn  func(ctx context.Context) ([]T, string, error)
}

// Exec simula a execução da consulta.
func (m *MockQueryBuilder[T]) Exec(ctx context.Context) ([]T, string, error) {
	if m.ExecFn != nil {
		return m.ExecFn(ctx)
	}
	return nil, "", nil
}

// Métodos fluentes vazios (só para compilar) que retornam o Builder mockado.
func (m *MockQueryBuilder[T]) Index(string) *QueryBuilder[T]            { return m.Builder }
func (m *MockQueryBuilder[T]) KeyEqual(string, any) *QueryBuilder[T]    { return m.Builder }
func (m *MockQueryBuilder[T]) FilterEqual(string, any) *QueryBuilder[T] { return m.Builder }
func (m *MockQueryBuilder[T]) Limit(int32) *QueryBuilder[T]             { return m.Builder }
func (m *MockQueryBuilder[T]) LastKey(string) *QueryBuilder[T]          { return m.Builder }
