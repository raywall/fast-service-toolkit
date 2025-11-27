// dyndb/types_test.go
package dyndb_test

// import (
// 	"encoding/base64"
// 	"encoding/json"
// 	"testing"

// 	"github.com/raywall/dynamodb-quick-service/dyndb"

// 	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
// 	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
// 	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// )

// func TestUnmarshalLogic(t *testing.T) {
// 	t.Parallel()

// 	// Testa a lógica de unmarshal que é usada internamente no QueryBuilder
// 	items := []map[string]types.AttributeValue{
// 		{
// 			"id":    &types.AttributeValueMemberS{Value: "123"},
// 			"name":  &types.AttributeValueMemberS{Value: "John"},
// 			"email": &types.AttributeValueMemberS{Value: "john@example.com"},
// 		},
// 	}

// 	lastKey := map[string]types.AttributeValue{
// 		"id": &types.AttributeValueMemberS{Value: "123"},
// 	}

// 	// Simula a lógica do Unmarshal
// 	result := make([]TestItem, 0, len(items))
// 	for _, item := range items {
// 		var testItem TestItem
// 		err := attributevalue.UnmarshalMap(item, &testItem)
// 		require.NoError(t, err) // Corrigido: usando 't' do teste, não 'testItem'
// 		result = append(result, testItem)
// 	}

// 	token := ""
// 	if lastKey != nil {
// 		if b, err := json.Marshal(lastKey); err == nil {
// 			token = base64.StdEncoding.EncodeToString(b)
// 		}
// 	}

// 	require.Len(t, result, 1)
// 	assert.Equal(t, "123", result[0].ID)
// 	assert.Equal(t, "John", result[0].Name)
// 	assert.NotEmpty(t, token)
// }

// func TestQueryFilter_Functions(t *testing.T) {
// 	t.Parallel()

// 	// Testa que as funções QueryFilter podem ser criadas sem erro
// 	keyCond := expression.KeyEqual(expression.Key("id"), expression.Value("123"))
// 	filterCond := expression.Equal(expression.Name("name"), expression.Value("John"))

// 	// Estas chamadas devem funcionar sem panic
// 	_ = dyndb.WithKeyCondition[TestItem](keyCond)
// 	_ = dyndb.WithFilter[TestItem](filterCond)
// 	_ = dyndb.WithIndex[TestItem]("test-index")
// 	_ = dyndb.WithLimit[TestItem](10)
// 	_ = dyndb.WithScanForward[TestItem](true)

// 	// Se chegamos aqui sem panic, o teste passa
// 	assert.True(t, true)
// }

// func TestErrNotFound(t *testing.T) {
// 	t.Parallel()
// 	assert.Equal(t, "dyndb: item not found", dyndb.ErrNotFound.Error())
// }

// func TestStoreInterface(t *testing.T) {
// 	t.Parallel()

// 	// Testa que nossa store implementa a interface Store
// 	client := &dyndb.MockDynamoClient{}
// 	cfg := dyndb.TableConfig[TestItem]{
// 		TableName: "test-table",
// 		HashKey:   "id",
// 	}

// 	store := dyndb.New(client, cfg)

// 	// Verifica se store implementa a interface Store[TestItem]
// 	var _ dyndb.Store[TestItem] = store

// 	// Testa que os métodos de query estão disponíveis
// 	queryBuilder := store.Query()
// 	assert.NotNil(t, queryBuilder)

// 	scanBuilder := store.Scan()
// 	assert.NotNil(t, scanBuilder)
// }

// func TestTableConfig(t *testing.T) {
// 	t.Parallel()

// 	// Testa que TableConfig pode ser criado corretamente
// 	cfg := dyndb.TableConfig[TestItem]{
// 		TableName:    "test-table",
// 		HashKey:      "id",
// 		SortKey:      "sk",
// 		TTLAttribute: "ttl",
// 	}

// 	assert.Equal(t, "test-table", cfg.TableName)
// 	assert.Equal(t, "id", cfg.HashKey)
// 	assert.Equal(t, "sk", cfg.SortKey)
// 	assert.Equal(t, "ttl", cfg.TTLAttribute)
// }

// func TestGlobalSecondaryIndex(t *testing.T) {
// 	t.Parallel()

// 	// Testa a estrutura GlobalSecondaryIndex
// 	gsi := dyndb.GlobalSecondaryIndex{
// 		Name:           "test-index",
// 		HashKey:        "gsi_pk",
// 		SortKey:        "gsi_sk",
// 		ProjectionType: types.ProjectionTypeAll,
// 	}

// 	assert.Equal(t, "test-index", gsi.Name)
// 	assert.Equal(t, "gsi_pk", gsi.HashKey)
// 	assert.Equal(t, "gsi_sk", gsi.SortKey)
// 	assert.Equal(t, types.ProjectionTypeAll, gsi.ProjectionType)
// }
