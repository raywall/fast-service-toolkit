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
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/raywall/fast-service-toolkit/tools/dyndb"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestStoreInterface(t *testing.T) {
	t.Parallel()

	// Testa que nossa store implementa a interface Store[TestItem]
	client := &MockDynamoClient{}
	cfg := dyndb.TableConfig[TestItem]{
		TableName: "test-table",
		HashKey:   "id",
	}

	store := dyndb.New(client, cfg)

	// Verifica se store implementa a interface Store[TestItem]
	var _ dyndb.Store[TestItem] = store

	// Testa que os métodos de query estão disponíveis
	queryBuilder := store.Query()
	assert.NotNil(t, queryBuilder)

	scanBuilder := store.Scan()
	assert.NotNil(t, scanBuilder)
}

func TestTableConfig(t *testing.T) {
	t.Parallel()

	// Testa que TableConfig pode ser criado corretamente
	cfg := dyndb.TableConfig[TestItem]{
		TableName:    "test-table",
		HashKey:      "id",
		SortKey:      "sk",
		TTLAttribute: "ttl",
	}

	assert.Equal(t, "test-table", cfg.TableName)
	assert.Equal(t, "id", cfg.HashKey)
	assert.Equal(t, "sk", cfg.SortKey)
	assert.Equal(t, "ttl", cfg.TTLAttribute)

	// Testa TableConfig sem SortKey e TTLAttribute
	cfgMinimal := dyndb.TableConfig[TestItem]{
		TableName: "minimal-table",
		HashKey:   "pk",
	}

	assert.Equal(t, "minimal-table", cfgMinimal.TableName)
	assert.Equal(t, "pk", cfgMinimal.HashKey)
	assert.Empty(t, cfgMinimal.SortKey)
	assert.Empty(t, cfgMinimal.TTLAttribute)
}

func TestGlobalSecondaryIndex(t *testing.T) {
	t.Parallel()

	// Testa a estrutura GlobalSecondaryIndex com diferentes projection types
	testCases := []struct {
		name         string
		gsi          dyndb.GlobalSecondaryIndex
		expectedName string
		expectedHash string
		expectedSort string
		expectedProj types.ProjectionType
	}{
		{
			name: "GSI with ALL projection",
			gsi: dyndb.GlobalSecondaryIndex{
				Name:           "test-index-all",
				HashKey:        "gsi_pk",
				SortKey:        "gsi_sk",
				ProjectionType: types.ProjectionTypeAll,
			},
			expectedName: "test-index-all",
			expectedHash: "gsi_pk",
			expectedSort: "gsi_sk",
			expectedProj: types.ProjectionTypeAll,
		},
		{
			name: "GSI with KEYS_ONLY projection",
			gsi: dyndb.GlobalSecondaryIndex{
				Name:           "test-index-keys",
				HashKey:        "pk",
				SortKey:        "sk",
				ProjectionType: types.ProjectionTypeKeysOnly,
			},
			expectedName: "test-index-keys",
			expectedHash: "pk",
			expectedSort: "sk",
			expectedProj: types.ProjectionTypeKeysOnly,
		},
		{
			name: "GSI with INCLUDE projection",
			gsi: dyndb.GlobalSecondaryIndex{
				Name:           "test-index-include",
				HashKey:        "hash",
				SortKey:        "sort",
				ProjectionType: types.ProjectionTypeInclude,
			},
			expectedName: "test-index-include",
			expectedHash: "hash",
			expectedSort: "sort",
			expectedProj: types.ProjectionTypeInclude,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectedName, tc.gsi.Name)
			assert.Equal(t, tc.expectedHash, tc.gsi.HashKey)
			assert.Equal(t, tc.expectedSort, tc.gsi.SortKey)
			assert.Equal(t, tc.expectedProj, tc.gsi.ProjectionType)
		})
	}
}

func TestQueryFilter_Functions(t *testing.T) {
	t.Parallel()

	// Testa que as funções QueryFilter podem ser criadas sem erro
	keyCond := expression.KeyEqual(expression.Key("id"), expression.Value("123"))
	filterCond := expression.Equal(expression.Name("name"), expression.Value("John"))

	// Estas chamadas devem funcionar sem panic
	keyFilter := dyndb.WithKeyCondition[TestItem](keyCond)
	filterFilter := dyndb.WithFilter[TestItem](filterCond)
	indexFilter := dyndb.WithIndex[TestItem]("test-index")
	limitFilter := dyndb.WithLimit[TestItem](10)
	scanForwardFilter := dyndb.WithScanForward[TestItem](true)
	lastKeyFilter := dyndb.WithLastEvaluatedKey[TestItem]("test-token")

	// Verifica que as funções foram criadas
	assert.NotNil(t, keyFilter)
	assert.NotNil(t, filterFilter)
	assert.NotNil(t, indexFilter)
	assert.NotNil(t, limitFilter)
	assert.NotNil(t, scanForwardFilter)
	assert.NotNil(t, lastKeyFilter)

	// Testa que são do tipo correto
	var _ dyndb.QueryFilter[TestItem] = keyFilter
	var _ dyndb.QueryFilter[TestItem] = filterFilter
	var _ dyndb.QueryFilter[TestItem] = indexFilter
	var _ dyndb.QueryFilter[TestItem] = limitFilter
	var _ dyndb.QueryFilter[TestItem] = scanForwardFilter
	var _ dyndb.QueryFilter[TestItem] = lastKeyFilter
}

func TestErrNotFound(t *testing.T) {
	t.Parallel()

	// Testa que ErrNotFound está definido corretamente
	assert.Equal(t, "dyndb: item not found", dyndb.ErrNotFound.Error())

	// Testa que é um erro
	var err error = dyndb.ErrNotFound
	assert.Error(t, err)
	assert.Equal(t, dyndb.ErrNotFound, err)
}

func TestQueryBuilder_Initialization(t *testing.T) {
	t.Parallel()

	// Testa que QueryBuilder é inicializado corretamente pelo store
	client := &MockDynamoClient{}
	store := createTestStore(client)

	// Test Query() initialization
	queryBuilder := store.Query()
	assert.NotNil(t, queryBuilder)

	// Test Scan() initialization
	scanBuilder := store.Scan()
	assert.NotNil(t, scanBuilder)
}

func TestUnmarshalLogic(t *testing.T) {
	t.Parallel()

	// Testa a lógica de unmarshal que é usada internamente no QueryBuilder
	testCases := []struct {
		name      string
		items     []map[string]types.AttributeValue
		lastKey   map[string]types.AttributeValue
		expected  []TestItem
		shouldErr bool
	}{
		{
			name: "Successful unmarshal with lastKey",
			items: []map[string]types.AttributeValue{
				{
					"id":    &types.AttributeValueMemberS{Value: "123"},
					"name":  &types.AttributeValueMemberS{Value: "John"},
					"email": &types.AttributeValueMemberS{Value: "john@example.com"},
				},
				{
					"id":    &types.AttributeValueMemberS{Value: "456"},
					"name":  &types.AttributeValueMemberS{Value: "Jane"},
					"email": &types.AttributeValueMemberS{Value: "jane@example.com"},
				},
			},
			lastKey: map[string]types.AttributeValue{
				"id": &types.AttributeValueMemberS{Value: "456"},
			},
			expected: []TestItem{
				{ID: "123", Name: "John", Email: "john@example.com"},
				{ID: "456", Name: "Jane", Email: "jane@example.com"},
			},
			shouldErr: false,
		},
		{
			name:      "Unmarshal empty items",
			items:     []map[string]types.AttributeValue{},
			lastKey:   nil,
			expected:  []TestItem{},
			shouldErr: false,
		},
		{
			name:      "Unmarshal with nil items",
			items:     nil,
			lastKey:   nil,
			expected:  []TestItem{},
			shouldErr: false,
		},
		{
			name: "Unmarshal without lastKey",
			items: []map[string]types.AttributeValue{
				{
					"id":   &types.AttributeValueMemberS{Value: "789"},
					"name": &types.AttributeValueMemberS{Value: "Bob"},
				},
			},
			lastKey: nil,
			expected: []TestItem{
				{ID: "789", Name: "Bob"},
			},
			shouldErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simula a lógica do Unmarshal
			result := make([]TestItem, 0, len(tc.items))
			for _, item := range tc.items {
				var testItem TestItem
				err := attributevalue.UnmarshalMap(item, &testItem)
				if tc.shouldErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				result = append(result, testItem)
			}

			token := ""
			if tc.lastKey != nil {
				// Para testar a serialização, vamos usar attributevalue.MarshalMap
				// que é a maneira correta de serializar AttributeValues
				avMap, err := attributevalue.MarshalMap(tc.lastKey)
				if err == nil {
					if b, err := json.Marshal(avMap); err == nil {
						token = base64.StdEncoding.EncodeToString(b)
					}
				}
			}

			assert.Len(t, result, len(tc.expected))
			for i, expectedItem := range tc.expected {
				assert.Equal(t, expectedItem.ID, result[i].ID)
				assert.Equal(t, expectedItem.Name, result[i].Name)
				assert.Equal(t, expectedItem.Email, result[i].Email)
			}

			if tc.lastKey != nil {
				assert.NotEmpty(t, token)
			} else {
				assert.Empty(t, token)
			}
		})
	}
}

func TestQueryFilter_Application(t *testing.T) {
	t.Parallel()

	// Testa que os QueryFilters podem ser aplicados a um QueryBuilder
	client := &MockDynamoClient{}
	store := createTestStore(client)

	// Mock para qualquer operação de Query
	client.On("Query", mock.Anything, mock.Anything).Return(&dynamodb.QueryOutput{
		Items: []map[string]types.AttributeValue{},
	}, nil)

	// Cria vários filtros
	keyCond := expression.KeyEqual(expression.Key("id"), expression.Value("123"))
	filterCond := expression.Equal(expression.Name("status"), expression.Value("active"))

	filters := []dyndb.QueryFilter[TestItem]{
		dyndb.WithKeyCondition[TestItem](keyCond),
		dyndb.WithFilter[TestItem](filterCond),
		dyndb.WithIndex[TestItem]("test-index"),
		dyndb.WithLimit[TestItem](5),
		dyndb.WithScanForward[TestItem](false),
	}

	// Aplica todos os filtros via Exec
	results, token, err := store.Query().Exec(context.Background(), filters...)

	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Empty(t, token)
	client.AssertExpectations(t)
}

func TestStoreMethodSignatures(t *testing.T) {
	t.Parallel()

	// Testa que a interface Store tem todos os métodos esperados
	client := &MockDynamoClient{}
	store := createTestStore(client)

	// Configura mocks para todas as chamadas esperadas

	// Mock para Get - retorna NotFound
	client.On("GetItem", mock.Anything, mock.Anything).Return(&dynamodb.GetItemOutput{
		Item: nil,
	}, nil)

	// Mock para Put
	client.On("PutItem", mock.Anything, mock.Anything).Return(&dynamodb.PutItemOutput{}, nil)

	// Mock para Delete
	client.On("DeleteItem", mock.Anything, mock.Anything).Return(&dynamodb.DeleteItemOutput{}, nil)

	// Mock para BatchWrite - CHAMADO APENAS SE HOUVER ITENS
	// Vamos passar alguns itens para garantir que seja chamado
	client.On("BatchWriteItem", mock.Anything, mock.Anything).Return(&dynamodb.BatchWriteItemOutput{}, nil)

	// Mock para BatchGet - CHAMADO APENAS SE HOUVER CHAVES
	// Vamos passar algumas chaves para garantir que seja chamado
	client.On("BatchGetItem", mock.Anything, mock.Anything).Return(&dynamodb.BatchGetItemOutput{
		Responses: map[string][]map[string]types.AttributeValue{},
	}, nil)

	// Testa assinatura do Get
	item, err := store.Get(context.Background(), "123", nil)
	assert.ErrorIs(t, err, dyndb.ErrNotFound)
	assert.Nil(t, item)

	// Testa assinatura do Put
	err = store.Put(context.Background(), TestItem{ID: "123"})
	assert.NoError(t, err)

	// Testa assinatura do Delete
	err = store.Delete(context.Background(), "123", nil)
	assert.NoError(t, err)

	// Testa assinatura do BatchWrite - COM ITENS para garantir chamada
	err = store.BatchWrite(context.Background(),
		[]TestItem{{ID: "1"}}, // Puts
		[][2]any{{"2", nil}},  // Deletes
	)
	assert.NoError(t, err)

	// Testa assinatura do BatchGet - COM CHAVES para garantir chamada
	items, err := store.BatchGet(context.Background(), [][2]any{{"1", nil}})
	assert.NoError(t, err)
	assert.Empty(t, items) // Mock retorna responses vazio

	client.AssertExpectations(t)
}

func TestComplexTypes(t *testing.T) {
	t.Parallel()

	// Testa com tipos complexos para verificar que genéricos funcionam
	type ComplexItem struct {
		ID      string            `dynamodbav:"id"`
		Data    map[string]string `dynamodbav:"data"`
		Numbers []int             `dynamodbav:"numbers"`
		Active  bool              `dynamodbav:"active"`
	}

	client := &MockDynamoClient{}
	cfg := dyndb.TableConfig[ComplexItem]{
		TableName: "complex-table",
		HashKey:   "id",
	}

	store := dyndb.New(client, cfg)

	// Verifica que a store foi criada com o tipo complexo
	var _ dyndb.Store[ComplexItem] = store

	// Testa métodos básicos
	queryBuilder := store.Query()
	assert.NotNil(t, queryBuilder)

	scanBuilder := store.Scan()
	assert.NotNil(t, scanBuilder)
}

func TestLastKeyTokenEncoding(t *testing.T) {
	t.Parallel()

	// Testa que o token é gerado corretamente quando há lastKey
	lastKey := map[string]types.AttributeValue{
		"id": &types.AttributeValueMemberS{Value: "123"},
	}

	// Gera token normalmente - sem verificação desnecessária
	b, err := json.Marshal(lastKey)
	require.NoError(t, err)
	token := base64.StdEncoding.EncodeToString(b)

	assert.NotEmpty(t, token)

	// Testa caso com lastKey nil
	var nilLastKey map[string]types.AttributeValue
	tokenFromNil := generateTokenFromLastKey(nilLastKey)
	assert.Empty(t, tokenFromNil)
}

// Função auxiliar para testar a lógica de geração de token
func generateTokenFromLastKey(lastKey map[string]types.AttributeValue) string {
	if lastKey != nil {
		b, err := json.Marshal(lastKey)
		if err != nil {
			return ""
		}
		return base64.StdEncoding.EncodeToString(b)
	}
	return ""
}
