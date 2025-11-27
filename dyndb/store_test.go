// dyndb/store_test.go
package dyndb_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGet_Success(t *testing.T) {
	t.Parallel()

	mockClient := &MockDynamoClient{}
	store := createTestStore(mockClient)

	expectedItem := map[string]types.AttributeValue{
		"id":    &types.AttributeValueMemberS{Value: "123"},
		"name":  &types.AttributeValueMemberS{Value: "John"},
		"email": &types.AttributeValueMemberS{Value: "john@example.com"},
	}

	mockClient.On("GetItem", mock.Anything, &dynamodb.GetItemInput{
		TableName:      aws.String("test-table"),
		Key:            map[string]types.AttributeValue{"id": &types.AttributeValueMemberS{Value: "123"}},
		ConsistentRead: aws.Bool(true),
	}).Return(&dynamodb.GetItemOutput{Item: expectedItem}, nil)

	item, err := store.Get(context.Background(), "123", nil)

	require.NoError(t, err)
	require.NotNil(t, item)
	assert.Equal(t, "123", item.ID)
	assert.Equal(t, "John", item.Name)
	mockClient.AssertExpectations(t)
}
