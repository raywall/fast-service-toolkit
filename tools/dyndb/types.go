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
package dyndb

import (
	"context"
	"errors"

	"encoding/base64"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// ErrNotFound – erro padrão retornado quando uma operação GetItem
// ou DeleteItem falha ao encontrar o item.
var ErrNotFound = errors.New("dyndb: item not found")

// DynamoDBClient interface para abstrair o cliente DynamoDB do SDK da AWS.
//
// Esta interface é usada internamente por `dynamoStore` e permite a substituição
// (mocking) do cliente real do DynamoDB.
type DynamoDBClient interface {
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	BatchWriteItem(ctx context.Context, params *dynamodb.BatchWriteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error)
	BatchGetItem(ctx context.Context, params *dynamodb.BatchGetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchGetItemOutput, error)
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
}

// Store — interface principal e genérica para interagir com o DynamoDB.
//
// O tipo genérico `T` é a struct Go que representa o item da tabela.
type Store[T any] interface {
	// Get item por chave primária (hashKey e sortKey opcional).
	Get(ctx context.Context, hashKey, sortKey any) (*T, error)
	// Put item (upsert). Adiciona o item à tabela.
	Put(ctx context.Context, item T) error
	// Delete item por chave primária.
	Delete(ctx context.Context, hashKey, sortKey any) error

	// BatchWrite realiza operações de Put e Delete em lote (máx. 25 operações).
	//   puts: Slice de itens a serem inseridos/atualizados.
	//   deletes: Slice de chaves [hashKey, sortKey] a serem deletadas.
	BatchWrite(ctx context.Context, puts []T, deletes [][2]any) error
	// BatchGet busca múltiplos itens por chave (máx. 100 chaves).
	//   keys: Slice de chaves [hashKey, sortKey] a serem buscadas.
	BatchGet(ctx context.Context, keys [][2]any) ([]T, error)

	// Query inicia o QueryBuilder[T] para construir uma consulta.
	Query() *QueryBuilder[T]
	// Scan inicia o QueryBuilder[T] para construir uma varredura (Scan).
	Scan() *QueryBuilder[T]
}

// GlobalSecondaryIndex contém a configuração para um GSI.
//
// As tags `env` permitem carregar a configuração de variáveis de ambiente.
type GlobalSecondaryIndex struct {
	Name    string `env:"DYNAMODB_GSI_NAME"`
	HashKey string `env:"DYNAMODB_GSI_HASH_KEY"`
	SortKey string `env:"DYNAMODB_GSI_SORT_KEY"`
	// ProjectionType define quais atributos são projetados no índice.
	ProjectionType types.ProjectionType `env:"DYNAMODB_GSI_PROJECTION_TYPE"`
}

// TableConfig — contém a configuração da tabela DynamoDB associada ao Store.
//
// O tipo genérico `T` é usado apenas para inferência e não armazena dados.
type TableConfig[T any] struct {
	TableName string `env:"DYNAMODB_TABLE_NAME"`
	HashKey   string `env:"DYNAMODB_HASH_KEY"`
	SortKey   string `env:"DYNAMODB_SORT_KEY"` // Opcional para tabelas com apenas HashKey
	// TTLAttribute é o nome do atributo de Time-To-Live.
	TTLAttribute string `env:"DYNAMODB_TTL_ATTRIBUTE"` // Opcional
}

// QueryFilter — tipo de função usado para aplicar filtros opcionais
// ao `QueryBuilder[T]` no momento da execução.
type QueryFilter[T any] func(*QueryBuilder[T])

// QueryBuilder — o builder fluente para operações Query e Scan.
//
// Contém o estado interno da consulta antes da sua execução.
type QueryBuilder[T any] struct {
	store       *dynamoStore[T]
	keyCond     *expression.KeyConditionBuilder
	filterCond  *expression.ConditionBuilder
	projection  *expression.ProjectionBuilder
	indexName   *string
	limit       *int32
	lastKey     map[string]types.AttributeValue
	scanForward *bool
	isScan      bool
}

// UnmarshalResults converte a lista de AttributeValues do DynamoDB para um slice
// de structs Go (`[]T`) e gera um token de paginação Base64.
//
// Parâmetros:
//
//	items: Lista de itens do DynamoDB.
//	lastKey: Chave da última avaliação retornada pelo DynamoDB.
//
// Retorna:
//
//	[]T: O slice de structs Go.
//	string: O token de paginação Base64 (string vazia se não houver mais itens).
//	error: Erro na desserialização (UnmarshalMap).
func (qb *QueryBuilder[T]) Unmarshal(items []map[string]types.AttributeValue, lastKey map[string]types.AttributeValue) ([]T, string, error) {
	result := make([]T, 0, len(items))
	for _, item := range items {
		var t T
		if err := attributevalue.UnmarshalMap(item, &t); err != nil {
			return nil, "", err
		}
		result = append(result, t)
	}
	token := ""
	if lastKey != nil {
		if b, err := json.Marshal(lastKey); err == nil {
			token = base64.StdEncoding.EncodeToString(b)
		}
	}
	return result, token, nil
}
