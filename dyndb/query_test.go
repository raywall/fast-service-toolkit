package dyndb_test

import (
	"context"
	"errors"
	"testing"

	"github.com/raywall/fast-service-toolkit/dyndb"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestQueryBuilder_FluentMethods(t *testing.T) {
	t.Parallel()

	mockClient := &MockDynamoClient{}
	store := createTestStore(mockClient)

	// Testa todos os métodos fluentes
	qb := store.Query().
		Index("test-index").
		KeyEqual("id", "123").
		KeyBeginsWith("name", "prefix").
		FilterEqual("email", "test@example.com").
		FilterContains("tags", "important").
		Limit(10).
		LastKey("test-token")

	assert.NotNil(t, qb)
}

func TestQuery_Exec_Success(t *testing.T) {
	t.Parallel()

	mockClient := &MockDynamoClient{}
	store := createTestStore(mockClient)

	expectedItems := []map[string]types.AttributeValue{
		{
			"id":    &types.AttributeValueMemberS{Value: "1"},
			"name":  &types.AttributeValueMemberS{Value: "Item1"},
			"email": &types.AttributeValueMemberS{Value: "item1@test.com"},
		},
		{
			"id":    &types.AttributeValueMemberS{Value: "2"},
			"name":  &types.AttributeValueMemberS{Value: "Item2"},
			"email": &types.AttributeValueMemberS{Value: "item2@test.com"},
		},
	}

	mockClient.On("Query", mock.Anything, mock.MatchedBy(func(input *dynamodb.QueryInput) bool {
		return *input.TableName == "test-table" &&
			input.KeyConditionExpression != nil
	})).Return(&dynamodb.QueryOutput{
		Items:            expectedItems,
		LastEvaluatedKey: nil,
	}, nil)

	results, token, err := store.Query().
		KeyEqual("id", "123").
		Exec(context.Background())

	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "1", results[0].ID)
	assert.Equal(t, "Item1", results[0].Name)
	assert.Equal(t, "2", results[1].ID)
	assert.Equal(t, "Item2", results[1].Name)
	assert.Empty(t, token)
	mockClient.AssertExpectations(t)
}

func TestQuery_Exec_WithIndex(t *testing.T) {
	t.Parallel()

	mockClient := &MockDynamoClient{}
	store := createTestStore(mockClient)

	expectedItems := []map[string]types.AttributeValue{
		{
			"id":   &types.AttributeValueMemberS{Value: "1"},
			"name": &types.AttributeValueMemberS{Value: "Item1"},
		},
	}

	mockClient.On("Query", mock.Anything, mock.MatchedBy(func(input *dynamodb.QueryInput) bool {
		return input.IndexName != nil && *input.IndexName == "test-index"
	})).Return(&dynamodb.QueryOutput{
		Items: expectedItems,
	}, nil)

	results, token, err := store.Query().
		Index("test-index").
		KeyEqual("id", "123").
		Exec(context.Background())

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "1", results[0].ID)
	assert.Empty(t, token)
	mockClient.AssertExpectations(t)
}

func TestQuery_Exec_WithPagination(t *testing.T) {
	t.Parallel()

	mockClient := &MockDynamoClient{}
	store := createTestStore(mockClient)

	expectedItems := []map[string]types.AttributeValue{
		{
			"id":   &types.AttributeValueMemberS{Value: "1"},
			"name": &types.AttributeValueMemberS{Value: "Item1"},
		},
	}

	lastKey := map[string]types.AttributeValue{
		"id": &types.AttributeValueMemberS{Value: "1"},
	}

	mockClient.On("Query", mock.Anything, mock.Anything).Return(&dynamodb.QueryOutput{
		Items:            expectedItems,
		LastEvaluatedKey: lastKey,
	}, nil)

	results, token, err := store.Query().
		KeyEqual("id", "123").
		Limit(1).
		Exec(context.Background())

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.NotEmpty(t, token) // Deve ter token devido ao LastEvaluatedKey
	mockClient.AssertExpectations(t)
}

func TestQuery_Exec_WithFilter(t *testing.T) {
	t.Parallel()

	mockClient := &MockDynamoClient{}
	store := createTestStore(mockClient)

	expectedItems := []map[string]types.AttributeValue{
		{
			"id":    &types.AttributeValueMemberS{Value: "1"},
			"name":  &types.AttributeValueMemberS{Value: "Item1"},
			"email": &types.AttributeValueMemberS{Value: "active@test.com"},
		},
	}

	mockClient.On("Query", mock.Anything, mock.MatchedBy(func(input *dynamodb.QueryInput) bool {
		return input.FilterExpression != nil
	})).Return(&dynamodb.QueryOutput{
		Items: expectedItems,
	}, nil)

	results, token, err := store.Query().
		KeyEqual("id", "123").
		FilterEqual("status", "active").
		Exec(context.Background())

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "1", results[0].ID)
	assert.Empty(t, token)
	mockClient.AssertExpectations(t)
}

func TestQuery_Exec_Error(t *testing.T) {
	t.Parallel()

	mockClient := &MockDynamoClient{}
	store := createTestStore(mockClient)

	expectedErr := errors.New("query error")
	mockClient.On("Query", mock.Anything, mock.Anything).Return(nil, expectedErr)

	results, token, err := store.Query().
		KeyEqual("id", "123").
		Exec(context.Background())

	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Empty(t, token)
	mockClient.AssertExpectations(t)
}

func TestScan_Exec_Success(t *testing.T) {
	t.Parallel()

	mockClient := &MockDynamoClient{}
	store := createTestStore(mockClient)

	expectedItems := []map[string]types.AttributeValue{
		{
			"id":    &types.AttributeValueMemberS{Value: "1"},
			"name":  &types.AttributeValueMemberS{Value: "Item1"},
			"email": &types.AttributeValueMemberS{Value: "item1@test.com"},
		},
		{
			"id":    &types.AttributeValueMemberS{Value: "2"},
			"name":  &types.AttributeValueMemberS{Value: "Item2"},
			"email": &types.AttributeValueMemberS{Value: "item2@test.com"},
		},
	}

	// Scan sem filtros - deve chamar ScanInput básico
	mockClient.On("Scan", mock.Anything, mock.MatchedBy(func(input *dynamodb.ScanInput) bool {
		return *input.TableName == "test-table" &&
			input.FilterExpression == nil && // Sem filtros
			input.ProjectionExpression == nil // Sem projeção
	})).Return(&dynamodb.ScanOutput{
		Items:            expectedItems,
		LastEvaluatedKey: nil,
	}, nil)

	results, token, err := store.Scan().
		Exec(context.Background())

	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "1", results[0].ID)
	assert.Equal(t, "2", results[1].ID)
	assert.Empty(t, token)
	mockClient.AssertExpectations(t)
}

func TestScan_Exec_WithFilter(t *testing.T) {
	t.Parallel()

	mockClient := &MockDynamoClient{}
	store := createTestStore(mockClient)

	expectedItems := []map[string]types.AttributeValue{
		{
			"id":   &types.AttributeValueMemberS{Value: "1"},
			"name": &types.AttributeValueMemberS{Value: "ActiveItem"},
		},
	}

	mockClient.On("Scan", mock.Anything, mock.MatchedBy(func(input *dynamodb.ScanInput) bool {
		return input.FilterExpression != nil // Deve ter filtro
	})).Return(&dynamodb.ScanOutput{
		Items: expectedItems,
	}, nil)

	results, token, err := store.Scan().
		FilterEqual("status", "active").
		Exec(context.Background())

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "1", results[0].ID)
	assert.Equal(t, "ActiveItem", results[0].Name)
	assert.Empty(t, token)
	mockClient.AssertExpectations(t)
}

func TestScan_Exec_WithLimit(t *testing.T) {
	t.Parallel()

	mockClient := &MockDynamoClient{}
	store := createTestStore(mockClient)

	expectedItems := []map[string]types.AttributeValue{
		{
			"id":   &types.AttributeValueMemberS{Value: "1"},
			"name": &types.AttributeValueMemberS{Value: "Item1"},
		},
	}

	mockClient.On("Scan", mock.Anything, mock.MatchedBy(func(input *dynamodb.ScanInput) bool {
		return input.Limit != nil && *input.Limit == 1 &&
			input.FilterExpression == nil // Apenas limite, sem filtros
	})).Return(&dynamodb.ScanOutput{
		Items: expectedItems,
	}, nil)

	results, token, err := store.Scan().
		Limit(1).
		Exec(context.Background())

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "1", results[0].ID)
	assert.Empty(t, token)
	mockClient.AssertExpectations(t)
}

func TestScan_Exec_Error(t *testing.T) {
	t.Parallel()

	mockClient := &MockDynamoClient{}
	store := createTestStore(mockClient)

	expectedErr := errors.New("scan error")

	// Mock para qualquer ScanInput
	mockClient.On("Scan", mock.Anything, mock.AnythingOfType("*dynamodb.ScanInput")).
		Return(nil, expectedErr)

	results, token, err := store.Scan().
		Exec(context.Background())

	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Empty(t, token)
	mockClient.AssertExpectations(t)
}

func TestQueryBuilder_LastKey(t *testing.T) {
	t.Parallel()

	mockClient := &MockDynamoClient{}
	store := createTestStore(mockClient)

	// Testa que LastKey não quebra com token vazio
	qb := store.Query().LastKey("")
	assert.NotNil(t, qb)

	// Testa com token inválido (não deve quebrar)
	qb = store.Query().LastKey("invalid-base64-token")
	assert.NotNil(t, qb)
}

func TestQueryFilter_Functions_Integration(t *testing.T) {
	t.Parallel()

	mockClient := &MockDynamoClient{}
	store := createTestStore(mockClient)

	expectedItems := []map[string]types.AttributeValue{
		{
			"id":   &types.AttributeValueMemberS{Value: "1"},
			"name": &types.AttributeValueMemberS{Value: "Item1"},
		},
	}

	mockClient.On("Query", mock.Anything, mock.Anything).Return(&dynamodb.QueryOutput{
		Items: expectedItems,
	}, nil)

	// Testa usando as funções QueryFilter
	keyCond := expression.KeyEqual(expression.Key("id"), expression.Value("123"))
	filterCond := expression.Equal(expression.Name("status"), expression.Value("active"))

	results, token, err := store.Query().
		Exec(context.Background(),
			dyndb.WithKeyCondition[TestItem](keyCond),
			dyndb.WithFilter[TestItem](filterCond),
			dyndb.WithIndex[TestItem]("test-index"),
			dyndb.WithLimit[TestItem](10),
		)

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "1", results[0].ID)
	assert.Empty(t, token)
	mockClient.AssertExpectations(t)
}

func TestQuery_WithComplexExpressions(t *testing.T) {
	t.Parallel()

	mockClient := &MockDynamoClient{}
	store := createTestStore(mockClient)

	expectedItems := []map[string]types.AttributeValue{
		{
			"id":    &types.AttributeValueMemberS{Value: "1"},
			"name":  &types.AttributeValueMemberS{Value: "Item1"},
			"email": &types.AttributeValueMemberS{Value: "test@example.com"},
			"age":   &types.AttributeValueMemberN{Value: "25"},
		},
	}

	mockClient.On("Query", mock.Anything, mock.MatchedBy(func(input *dynamodb.QueryInput) bool {
		return input.KeyConditionExpression != nil &&
			input.FilterExpression != nil
	})).Return(&dynamodb.QueryOutput{
		Items: expectedItems,
	}, nil)

	// Testa expressões complexas combinadas
	results, token, err := store.Query().
		KeyEqual("pk", "partition1").
		KeyBeginsWith("sk", "prefix").
		FilterEqual("status", "active").
		FilterContains("tags", "important").
		Limit(10).
		Exec(context.Background())

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "1", results[0].ID)
	assert.Empty(t, token)
	mockClient.AssertExpectations(t)
}

func TestScan_WithPagination(t *testing.T) {
	t.Parallel()

	mockClient := &MockDynamoClient{}
	store := createTestStore(mockClient)

	expectedItems := []map[string]types.AttributeValue{
		{
			"id":   &types.AttributeValueMemberS{Value: "1"},
			"name": &types.AttributeValueMemberS{Value: "Item1"},
		},
	}

	lastKey := map[string]types.AttributeValue{
		"id": &types.AttributeValueMemberS{Value: "1"},
	}

	mockClient.On("Scan", mock.Anything, mock.MatchedBy(func(input *dynamodb.ScanInput) bool {
		return input.ExclusiveStartKey != nil || // Com last key
			input.Limit != nil // Ou com limite
	})).Return(&dynamodb.ScanOutput{
		Items:            expectedItems,
		LastEvaluatedKey: lastKey,
	}, nil)

	results, token, err := store.Scan().
		Limit(1).
		Exec(context.Background())

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.NotEmpty(t, token)
	mockClient.AssertExpectations(t)
}

// Teste adicional para Scan sem condições
func TestScan_Exec_NoConditions(t *testing.T) {
	t.Parallel()

	mockClient := &MockDynamoClient{}
	store := createTestStore(mockClient)

	expectedItems := []map[string]types.AttributeValue{
		{
			"id":   &types.AttributeValueMemberS{Value: "1"},
			"name": &types.AttributeValueMemberS{Value: "Item1"},
		},
	}

	// Scan sem nenhuma condição - apenas table name
	mockClient.On("Scan", mock.Anything, &dynamodb.ScanInput{
		TableName: aws.String("test-table"),
	}).Return(&dynamodb.ScanOutput{
		Items: expectedItems,
	}, nil)

	results, token, err := store.Scan().Exec(context.Background())

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "1", results[0].ID)
	assert.Empty(t, token)
	mockClient.AssertExpectations(t)
}
