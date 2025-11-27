// dyndb/mock_test.go
package dyndb_test

// import (
// 	"github.com/raywall/dynamodb-quick-service/dyndb"
// )

// var mockDynamoClient = &dyndb.MockDynamoClient{}

// // TestItem é uma estrutura de teste para os testes
// type TestItem struct {
// 	ID    string `dynamodbav:"id"`
// 	Name  string `dynamodbav:"name"`
// 	Email string `dynamodbav:"email"`
// 	TTL   int64  `dynamodbav:"ttl,omitempty"`
// }

// // TestItemWithSortKey é uma estrutura com chave de ordenação
// type TestItemWithSortKey struct {
// 	PK   string `dynamodbav:"pk"`
// 	SK   string `dynamodbav:"sk"`
// 	Data string `dynamodbav:"data"`
// }

// // helper function para criar store de teste
// func createTestStore(client *dyndb.MockDynamoClient) dyndb.Store[TestItem] {
// 	cfg := dyndb.TableConfig[TestItem]{
// 		TableName: "test-table",
// 		HashKey:   "id",
// 	}
// 	// Corrigido: convertendo MockDynamoClient para *dynamodb.Client
// 	// Como MockDynamoClient implementa a mesma interface, podemos usar uma conversão
// 	return dyndb.New[TestItem](mockDynamoClient, cfg)
// }

// // helper function para criar store com sort key
// func createTestStoreWithSortKey(client *dyndb.MockDynamoClient) dyndb.Store[TestItemWithSortKey] {
// 	cfg := dyndb.TableConfig[TestItemWithSortKey]{
// 		TableName: "test-table",
// 		HashKey:   "pk",
// 		SortKey:   "sk",
// 	}
// 	return dyndb.New[TestItemWithSortKey](mockDynamoClient, cfg)
// }
