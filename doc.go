// Package dynamodb_quick_service fornece um conjunto de utilitários e abstrações
// para acelerar o desenvolvimento de serviços backend em Go, focados em
// operações robustas com DynamoDB, carregamento de configuração e gerenciamento
// de microserviços.
//
// Visão Geral:
// Este módulo é uma caixa de ferramentas para construir aplicações de forma
// rápida e eficiente, fornecendo soluções modulares para:
// 1. Configuração (envloader): Carregamento de variáveis de ambiente para structs.
// 2. Persistência de Dados (dyndb): Camada genérica, tipada e fluente sobre DynamoDB.
// 3. Orquestração (api): Pipeline concorrente para chamadas de APIs externas e STS.
//
// O design é focado na composabilidade e testabilidade, utilizando interfaces
// e genéricos para garantir tipagem segura e fácil mocking.
//
// Sub-Pacotes Principais:
//
// 1. envloader:
//   - Carregamento de configurações via tags "env" e "envDefault".
//   - Suporte a tipos nativos e structs aninhadas, com tratamento de erros tipados.
//
// 2. dyndb:
//   - Abstração de persistência (Store[T]).
//   - CRUD tipado e operações Batch.
//   - QueryBuilder para consultas complexas com paginação segura.
//
// 3. api:
//   - APIPipeline para execução concorrente de chamadas HTTP com gerenciamento de dependências.
//   - Service Token Service (TokenService) para autenticação centralizada.
//
// Exemplo de Início Rápido:
//
// Demonstração da combinação de envloader e dyndb para inicialização de um Store.
//
//	package main
//
//	import (
//		"context"
//		"log"
//
//		"github.com/aws/aws-sdk-go-v2/aws"
//		"github.com/aws/aws-sdk-go-v2/service/dynamodb" // Cliente AWS
//		"github.com/raywall/fast-service-toolkit/dyndb"
//		"github.com/raywall/fast-service-toolkit/envloader"
//	)
//
//	// Estrutura para ser preenchida pelo envloader
//	type AppConfig struct {
//		TableName string `env:"DYNAMODB_TABLE_NAME"`
//		HashKey string `env:"DYNAMODB_HASH_KEY" envDefault:"id"`
//	}
//
//	type User struct {
//		ID string `dynamodbav:"id"`
//		Name string `dynamodbav:"name"`
//	}
//
//	func main() {
//		// 1. Carregar configuração usando envloader (Foco no envloader!)
//		var cfg AppConfig
//		if err := envloader.Load(&cfg); err != nil {
//			log.Fatalf("Erro ao carregar env: %v", err)
//		}
//
//		// 2. Criar a configuração do Store tipado
//		tableCfg := dyndb.TableConfig[User]{
//			TableName: cfg.TableName,
//			HashKey:   cfg.HashKey,
//		}
//
//		// 3. Inicializar Store
//		// client := dynamodb.NewFromConfig(awsConfig) // Assumindo awsConfig configurado
//		client := &dyndb.MockDynamoClient{} // Usando mock para o exemplo simples
//		userStore := dyndb.New(client, tableCfg)
//
//		// 4. Usar o Store
//		item, err := userStore.Get(context.Background(), "user-123", nil)
//		if err != nil && err != dyndb.ErrNotFound {
//			log.Fatalf("Erro ao buscar item: %v", err)
//		}
//		log.Printf("Item: %v", item)
//	}
package dynamodb_quick_service
