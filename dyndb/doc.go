// Package dyndb fornece uma abstração genérica e fortemente tipada sobre o
// AWS DynamoDB Go SDK (v2).
//
// Visão Geral:
// O pacote `dyndb` oferece a interface `Store[T]`, que simplifica as operações
// CRUD e Batch, eliminando a necessidade de lidar diretamente com os tipos
// de baixo nível do SDK do DynamoDB (AttributeValue, etc.).
//
// A principal característica é o `QueryBuilder[T]`, que permite construir
// consultas (`Query` e `Scan`) complexas de forma fluente e segura em tempo
// de compilação, abstraindo as Expression Builders do SDK.
//
// Funcionalidades Principais:
// - CRUD Tipado: Operações `Get`, `Put`, `Delete` usando tipos Go nativos.
// - Batch Otimizado: Suporte a `BatchWrite` (puts e deletes) e `BatchGet`.
// - Builder Fluente: `Query().KeyEqual(...).FilterEqual(...).Exec(...)` para consultas.
// - Paginação Automática: Conversão de `LastEvaluatedKey` em tokens Base64 para paginação.
// - Mocks Integrados: `MockStore` e `MockDynamoClient` para testes unitários fáceis.
//
// Exemplos de Uso:
//
// Exemplo Básico de Store e CRUD:
// Demonstra como criar o Store e realizar operações básicas.
//
//	type User struct {
//		ID string `dynamodbav:"id"`
//		Email string `dynamodbav:"email"`
//	}
//
//	// Configuração do cliente e tabela
//	cfg := dyndb.TableConfig[User]{TableName: "Users", HashKey: "id"}
//	client := &MockDynamoClient{} // Use o cliente real do SDK em produção
//
//	userStore := dyndb.New(client, cfg)
//
//	// Operação Put
//	userStore.Put(context.Background(), User{ID: "u1", Email: "a@b.com"})
//
//	// Operação Get
//	user, err := userStore.Get(context.Background(), "u1", nil)
//	if err == dyndb.ErrNotFound { /* ... */ }
//
// Exemplo de Query Fluente:
//
//	results, token, err := userStore.Query().
//		Index("GSI1").
//		KeyEqual("GSI1PK", "some_value").
//		FilterEqual("status", "ACTIVE").
//		Limit(50).
//		Exec(context.Background())
//
// Configuração:
// O Store é configurado via `TableConfig[T]` ou usando variáveis de ambiente
// para a configuração da tabela (HashKey, SortKey, etc.).
package dyndb
