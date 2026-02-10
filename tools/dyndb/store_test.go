// Copyright 2025 Raywall Malheiros de Souza
// Licensed under the Mozilla Public License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.mozilla.org/en-US/MPL/2.0/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
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
