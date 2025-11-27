// dyndb/query_test.go
package dyndb_test

// import (
// 	"context"
// 	"errors"
// 	"testing"

// 	"github.com/raywall/dynamodb-quick-service/dyndb"

// 	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
// 	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/mock"
// 	"github.com/stretchr/testify/require"
// )

// func TestQueryBuilder_Methods(t *testing.T) {
// 	t.Parallel()

// 	mockClient := &MockDynamoClient{}
// 	store := createTestStore(mockClient)

// 	qb := store.Query().
// 		Index("test-index").
// 		KeyEqual("id", "123").
// 		KeyBeginsWith("name", "prefix").
// 		FilterEqual("email", "test@example.com").
// 		FilterContains("tags", "important").
// 		Limit(10).
// 		LastKey("")

// 	assert.NotNil(t, qb)
	
// 	// Conversão para acessar campos internos
// 	queryBuilder := qb.(*dyndb.QueryBuilder[TestItem])
// 	assert.Equal(t, "test-index", *queryBuilder.IndexName())
// 	assert.NotNil(t, queryBuilder.KeyCondition())
// 	assert.NotNil(t, queryBuilder.FilterCondition())
// 	assert.Equal(t, int32(10), *queryBuilder.Limit())
// }

// func TestQuery_Exec_Success(t *testing.T) {
// 	t.Parallel()

// 	mockClient := &MockDynamoClient{}
// 	store := createTestStore(mockClient)

// 	expectedItems := []map[string]types.AttributeValue{
// 		{
// 			"id":    &types.AttributeValueMemberS{Value: "1"},
// 			"name":  &types.AttributeValueMemberS{Value: "Item1"},
// 			"email": &types.AttributeValueMemberS{Value: "item1@test.com"},
// 		},
// 		{
// 			"id":    &types.AttributeValueMemberS{Value: "2"},
// 			"name":  &types.AttributeValueMemberS{Value: "Item2"},
// 			"email": &types.AttributeValueMemberS{Value: "item2@test.com"},
// 		},
// 	}

// 	mockClient.On("Query", mock.Anything, mock.MatchedBy(func(input *dynamodb.QueryInput) bool {
// 		return *input.TableName == "test-table" &&
// 			input.KeyConditionExpression != nil
// 	})).Return(&dynamodb.QueryOutput{
// 		Items:            expectedItems,
// 		LastEvaluatedKey: nil,
// 	}, nil)

// 	qb := store.Query().KeyEqual("id", "123")
// 	results, token, err := qb.Exec(context.Background())

// 	require.NoError(t, err)
// 	require.Len(t, results, 2)
// 	assert.Equal(t, "1", results[0].ID)
// 	assert.Equal(t, "2", results[1].ID)
// 	assert.Empty(t, token)
// 	mockClient.AssertExpectations(t)
// }

// func TestScan_Exec_Success(t *testing.T) {
// 	t.Parallel()

// 	mockClient := &MockDynamoClient{}
// 	store := createTestStore(mockClient)

// 	expectedItems := []map[string]types.AttributeValue{
// 		{
// 			"id":    &types.AttributeValueMemberS{Value: "1"},
// 			"name":  &types.AttributeValueMemberS{Value: "Item1"},
// 			"email": &types.AttributeValueMemberS{Value: "item1@test.com"},
// 		},
// 	}

// 	mockClient.On("Scan", mock.Anything, mock.MatchedBy(func(input *dynamodb.ScanInput) bool {
// 		return *input.TableName == "test-table"
// 	})).Return(&dynamodb.ScanOutput{
// 		Items:            expectedItems,
// 		LastEvaluatedKey: nil,
// 	}, nil)

// 	qb := store.Scan().FilterEqual("name", "Item1")
// 	results, token, err := qb.Exec(context.Background())

// 	require.NoError(t, err)
// 	require.Len(t, results, 1)
// 	assert.Equal(t, "1", results[0].ID)
// 	assert.Empty(t, token)
// 	mockClient.AssertExpectations(t)
// }

// func TestQuery_Exec_WithPagination(t *testing.T) {
// 	t.Parallel()

// 	mockClient := &MockDynamoClient{}
// 	store := createTestStore(mockClient)

// 	expectedItems := []map[string]types.AttributeValue{
// 		{
// 			"id":    &types.AttributeValueMemberS{Value: "1"},
// 			"name":  &types.AttributeValueMemberS{Value: "Item1"},
// 			"email": &types.AttributeValueMemberS{Value: "item1@test.com"},
// 		},
// 	}

// 	lastKey := map[string]types.AttributeValue{
// 		"id": &types.AttributeValueMemberS{Value: "1"},
// 	}

// 	mockClient.On("Query", mock.Anything, mock.Anything).Return(&dynamodb.QueryOutput{
// 		Items:            expectedItems,
// 		LastEvaluatedKey: lastKey,
// 	}, nil)

// 	qb := store.Query().KeyEqual("id", "123")
// 	results, token, err := qb.Exec(context.Background())

// 	require.NoError(t, err)
// 	require.Len(t, results, 1)
// 	assert.NotEmpty(t, token) // Deve ter token devido ao LastEvaluatedKey
// 	mockClient.AssertExpectations(t)
// }

// func TestQuery_Exec_Error(t *testing.T) {
// 	t.Parallel()

// 	mockClient := &MockDynamoClient{}
// 	store := createTestStore(mockClient)

// 	expectedErr := errors.New("query error")
// 	mockClient.On("Query", mock.Anything, mock.Anything).Return(nil, expectedErr)

// 	qb := store.Query().KeyEqual("id", "123")
// 	results, token, err := qb.Exec(context.Background())

// 	assert.Error(t, err)
// 	assert.Nil(t, results)
// 	assert.Empty(t, token)
// }

// func TestScan_Exec_Error(t *testing.T) {
// 	t.Parallel()

// 	mockClient := &MockDynamoClient{}
// 	store := createTestStore(mockClient)

// 	expectedErr := errors.New("scan error")
// 	mockClient.On("Scan", mock.Anything, mock.Anything).Return(nil, expectedErr)

// 	qb := store.Scan()
// 	results, token, err := qb.Exec(context.Background())

// 	assert.Error(t, err)
// 	assert.Nil(t, results)
// 	assert.Empty(t, token)
// }

// func TestQueryBuilder_FluentInterface(t *testing.T) {
// 	t.Parallel()

// 	mockClient := &MockDynamoClient{}
// 	store := createTestStore(mockClient)

// 	// Testa a interface fluente completa
// 	qb := store.Query().
// 		Index("test-index").
// 		KeyEqual("partitionKey", "value1").
// 		KeyBeginsWith("sortKey", "prefix").
// 		FilterEqual("status", "active").
// 		FilterContains("description", "important").
// 		Limit(5).
// 		LastKey("")

// 	assert.NotNil(t, qb)
// }

// func TestMockStore_Implementation(t *testing.T) {
// 	t.Parallel()

// 	mockStore := &dyndb.MockStore[TestItem]{
// 		GetFn: func(ctx context.Context, hashKey, sortKey any) (*TestItem, error) {
// 			return &TestItem{ID: "123", Name: "Test"}, nil
// 		},
// 		PutFn: func(ctx context.Context, item TestItem) error {
// 			return nil
// 		},
// 		DeleteFn: func(ctx context.Context, hashKey, sortKey any) error {
// 			return nil
// 		},
// 	}

// 	// Test Get
// 	item, err := mockStore.Get(context.Background(), "123", nil)
// 	require.NoError(t, err)
// 	assert.Equal(t, "123", item.ID)

// 	// Test Put
// 	err = mockStore.Put(context.Background(), TestItem{ID: "123"})
// 	assert.NoError(t, err)

// 	// Test Delete
// 	err = mockStore.Delete(context.Background(), "123", nil)
// 	assert.NoError(t, err)

// 	// Test Query (default implementation)
// 	queryBuilder := mockStore.Query()
// 	assert.NotNil(t, queryBuilder)

// 	// Test Scan (default implementation)
// 	scanBuilder := mockStore.Scan()
// 	assert.NotNil(t, scanBuilder)
// }