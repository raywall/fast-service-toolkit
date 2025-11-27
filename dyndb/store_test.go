// dyndb/store_test.go
package dyndb_test

// import (
// 	"context"
// 	"errors"
// 	"testing"

// 	"github.com/raywall/dynamodb-quick-service/dyndb"

// 	"github.com/aws/aws-sdk-go-v2/aws"
// 	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
// 	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/mock"
// 	"github.com/stretchr/testify/require"
// )

// func TestNew(t *testing.T) {
// 	t.Parallel()

// 	mockClient := &dyndb.MockDynamoClient{}
// 	cfg := dyndb.TableConfig[TestItem]{
// 		TableName: "test-table",
// 		HashKey:   "id",
// 	}

// 	store := dyndb.New(mockClient, cfg)

// 	assert.NotNil(t, store)
// }

// func TestGet_Success(t *testing.T) {
// 	t.Parallel()

// 	mockClient := &dyndb.MockDynamoClient{}
// 	store := createTestStore(mockClient)

// 	expectedItem := map[string]types.AttributeValue{
// 		"id":    &types.AttributeValueMemberS{Value: "123"},
// 		"name":  &types.AttributeValueMemberS{Value: "John"},
// 		"email": &types.AttributeValueMemberS{Value: "john@example.com"},
// 	}

// 	mockClient.GetItem(context.Background(), &dynamodb.GetItemInput{
// 		TableName:      aws.String("test-table"),
// 		Key:            map[string]types.AttributeValue{"id": &types.AttributeValueMemberS{Value: "123"}},
// 		ConsistentRead: aws.Bool(true),
// 	}).Return(&dynamodb.GetItemOutput{Item: expectedItem}, nil)

// 	item, err := store.Get(context.Background(), "123", nil)

// 	require.NoError(t, err)
// 	require.NotNil(t, item)
// 	assert.Equal(t, "123", item.ID)
// 	assert.Equal(t, "John", item.Name)
// 	mockClient.AssertExpectations(t)
// }

// func TestGet_WithSortKey(t *testing.T) {
// 	t.Parallel()

// 	mockClient := &dyndb.MockDynamoClient{}
// 	store := createTestStoreWithSortKey(mockClient)

// 	expectedItem := map[string]types.AttributeValue{
// 		"pk":   &types.AttributeValueMemberS{Value: "partition1"},
// 		"sk":   &types.AttributeValueMemberS{Value: "sort1"},
// 		"data": &types.AttributeValueMemberS{Value: "test data"},
// 	}

// 	mockClient.On("GetItem", mock.Anything, &dynamodb.GetItemInput{
// 		TableName: aws.String("test-table"),
// 		Key: map[string]types.AttributeValue{
// 			"pk": &types.AttributeValueMemberS{Value: "partition1"},
// 			"sk": &types.AttributeValueMemberS{Value: "sort1"},
// 		},
// 		ConsistentRead: aws.Bool(true),
// 	}).Return(&dynamodb.GetItemOutput{Item: expectedItem}, nil)

// 	item, err := store.Get(context.Background(), "partition1", "sort1")

// 	require.NoError(t, err)
// 	require.NotNil(t, item)
// 	assert.Equal(t, "partition1", item.PK)
// 	assert.Equal(t, "sort1", item.SK)
// 	assert.Equal(t, "test data", item.Data)
// }

// func TestGet_NotFound(t *testing.T) {
// 	t.Parallel()

// 	mockClient := &dyndb.MockDynamoClient{}
// 	store := createTestStore(mockClient)

// 	mockClient.On("GetItem", mock.Anything, mock.Anything).Return(&dynamodb.GetItemOutput{Item: nil}, nil)

// 	item, err := store.Get(context.Background(), "123", nil)

// 	assert.ErrorIs(t, err, dyndb.ErrNotFound)
// 	assert.Nil(t, item)
// }

// func TestGet_Error(t *testing.T) {
// 	t.Parallel()

// 	mockClient := &dyndb.MockDynamoClient{}
// 	store := createTestStore(mockClient)

// 	expectedErr := errors.New("dynamodb error")
// 	mockClient.On("GetItem", mock.Anything, mock.Anything).Return(nil, expectedErr)

// 	item, err := store.Get(context.Background(), "123", nil)

// 	assert.ErrorContains(t, err, "dynamostore: get failed:")
// 	assert.ErrorContains(t, err, "dynamodb error")
// 	assert.Nil(t, item)
// }

// func TestPut_Success(t *testing.T) {
// 	t.Parallel()

// 	mockClient := &dyndb.MockDynamoClient{}
// 	store := createTestStore(mockClient)

// 	testItem := TestItem{
// 		ID:    "123",
// 		Name:  "John",
// 		Email: "john@example.com",
// 	}

// 	mockClient.On("PutItem", mock.Anything, mock.MatchedBy(func(input *dynamodb.PutItemInput) bool {
// 		return *input.TableName == "test-table" &&
// 			input.Item["id"].(*types.AttributeValueMemberS).Value == "123"
// 	})).Return(&dynamodb.PutItemOutput{}, nil)

// 	err := store.Put(context.Background(), testItem)

// 	assert.NoError(t, err)
// 	mockClient.AssertExpectations(t)
// }

// func TestPut_WithTTL(t *testing.T) {
// 	t.Parallel()

// 	mockClient := &dyndb.MockDynamoClient{}
// 	cfg := dyndb.TableConfig[TestItem]{
// 		TableName:    "test-table",
// 		HashKey:      "id",
// 		TTLAttribute: "ttl",
// 	}
// 	store := dyndb.New(mockClient, cfg)

// 	testItem := TestItem{
// 		ID:    "123",
// 		Name:  "John",
// 		Email: "john@example.com",
// 		// TTL não definido - deve ser preenchido automaticamente
// 	}

// 	mockClient.On("PutItem", mock.Anything, mock.MatchedBy(func(input *dynamodb.PutItemInput) bool {
// 		ttlAttr := input.Item["ttl"]
// 		if ttlAttr == nil {
// 			return false
// 		}
// 		// Verifica se o TTL foi definido (deve ser um timestamp futuro)
// 		ttlValue := ttlAttr.(*types.AttributeValueMemberN).Value
// 		return ttlValue != "" && len(ttlValue) > 0
// 	})).Return(&dynamodb.PutItemOutput{}, nil)

// 	err := store.Put(context.Background(), testItem)

// 	assert.NoError(t, err)
// 	mockClient.AssertExpectations(t)
// }

// func TestPut_Error(t *testing.T) {
// 	t.Parallel()

// 	mockClient := &dyndb.MockDynamoClient{}
// 	store := createTestStore(mockClient)

// 	testItem := TestItem{ID: "123"}
// 	expectedErr := errors.New("put error")

// 	mockClient.On("PutItem", mock.Anything, mock.Anything).Return(nil, expectedErr)

// 	err := store.Put(context.Background(), testItem)

// 	assert.ErrorContains(t, err, "dynamostore: put failed:")
// 	assert.ErrorContains(t, err, "put error")
// }

// func TestDelete_Success(t *testing.T) {
// 	t.Parallel()

// 	mockClient := &dyndb.MockDynamoClient{}
// 	store := createTestStore(mockClient)

// 	mockClient.On("DeleteItem", mock.Anything, &dynamodb.DeleteItemInput{
// 		TableName: aws.String("test-table"),
// 		Key:       map[string]types.AttributeValue{"id": &types.AttributeValueMemberS{Value: "123"}},
// 	}).Return(&dynamodb.DeleteItemOutput{}, nil)

// 	err := store.Delete(context.Background(), "123", nil)

// 	assert.NoError(t, err)
// 	mockClient.AssertExpectations(t)
// }

// func TestDelete_WithSortKey(t *testing.T) {
// 	t.Parallel()

// 	mockClient := &dyndb.MockDynamoClient{}
// 	store := createTestStoreWithSortKey(mockClient)

// 	mockClient.On("DeleteItem", mock.Anything, &dynamodb.DeleteItemInput{
// 		TableName: aws.String("test-table"),
// 		Key: map[string]types.AttributeValue{
// 			"pk": &types.AttributeValueMemberS{Value: "partition1"},
// 			"sk": &types.AttributeValueMemberS{Value: "sort1"},
// 		},
// 	}).Return(&dynamodb.DeleteItemOutput{}, nil)

// 	err := store.Delete(context.Background(), "partition1", "sort1")

// 	assert.NoError(t, err)
// 	mockClient.AssertExpectations(t)
// }

// func TestBatchWrite_Success(t *testing.T) {
// 	t.Parallel()

// 	mockClient := &dyndb.MockDynamoClient{}
// 	store := createTestStore(mockClient)

// 	puts := []TestItem{
// 		{ID: "1", Name: "Item1"},
// 		{ID: "2", Name: "Item2"},
// 	}
// 	deletes := [][2]any{
// 		{"3", nil},
// 		{"4", nil},
// 	}

// 	// Espera uma chamada com 4 itens (2 puts + 2 deletes)
// 	mockClient.On("BatchWriteItem", mock.Anything, mock.MatchedBy(func(input *dynamodb.BatchWriteItemInput) bool {
// 		requests := input.RequestItems["test-table"]
// 		return len(requests) == 4
// 	})).Return(&dynamodb.BatchWriteItemOutput{}, nil)

// 	err := store.BatchWrite(context.Background(), puts, deletes)

// 	assert.NoError(t, err)
// 	mockClient.AssertExpectations(t)
// }

// func TestBatchWrite_LargeBatch(t *testing.T) {
// 	t.Parallel()

// 	mockClient := &dyndb.MockDynamoClient{}
// 	store := createTestStore(mockClient)

// 	// Cria 30 itens para testar a divisão em lotes
// 	puts := make([]TestItem, 30)
// 	for i := 0; i < 30; i++ {
// 		puts[i] = TestItem{ID: string(rune('A' + i)), Name: "Item"}
// 	}

// 	// Espera 2 chamadas (25 + 5)
// 	mockClient.On("BatchWriteItem", mock.Anything, mock.MatchedBy(func(input *dynamodb.BatchWriteItemInput) bool {
// 		return len(input.RequestItems["test-table"]) == 25
// 	})).Return(&dynamodb.BatchWriteItemOutput{}, nil)

// 	mockClient.On("BatchWriteItem", mock.Anything, mock.MatchedBy(func(input *dynamodb.BatchWriteItemInput) bool {
// 		return len(input.RequestItems["test-table"]) == 5
// 	})).Return(&dynamodb.BatchWriteItemOutput{}, nil)

// 	err := store.BatchWrite(context.Background(), puts, nil)

// 	assert.NoError(t, err)
// 	mockClient.AssertExpectations(t)
// }

// func TestBatchGet_Success(t *testing.T) {
// 	t.Parallel()

// 	mockClient := &dyndb.MockDynamoClient{}
// 	store := createTestStore(mockClient)

// 	keys := [][2]any{
// 		{"1", nil},
// 		{"2", nil},
// 	}

// 	expectedItems := []map[string]types.AttributeValue{
// 		{
// 			"id":   &types.AttributeValueMemberS{Value: "1"},
// 			"name": &types.AttributeValueMemberS{Value: "Item1"},
// 		},
// 		{
// 			"id":   &types.AttributeValueMemberS{Value: "2"},
// 			"name": &types.AttributeValueMemberS{Value: "Item2"},
// 		},
// 	}

// 	mockClient.On("BatchGetItem", mock.Anything, mock.Anything).Return(&dynamodb.BatchGetItemOutput{
// 		Responses: map[string][]map[string]types.AttributeValue{
// 			"test-table": expectedItems,
// 		},
// 	}, nil)

// 	results, err := store.BatchGet(context.Background(), keys)

// 	require.NoError(t, err)
// 	require.Len(t, results, 2)
// 	assert.Equal(t, "1", results[0].ID)
// 	assert.Equal(t, "2", results[1].ID)
// 	mockClient.AssertExpectations(t)
// }
