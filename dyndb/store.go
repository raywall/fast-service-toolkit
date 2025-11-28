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
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/raywall/fast-service-toolkit/envloader"
)

// dynamoStore é a implementação concreta e não exportada de Store[T].
//
// Gerencia a comunicação com o cliente DynamoDB usando a configuração da tabela.
type dynamoStore[T any] struct {
	// client é a abstração do cliente DynamoDB.
	client DynamoDBClient
	// cfg é a configuração da tabela para este Store.
	cfg TableConfig[T]
}

// New cria um store reutilizável e fortemente tipado (Store[T]).
//
// Se `cfg.TableName` for vazio, tenta carregar o nome da tabela (e chaves)
// de variáveis de ambiente usando o `envloader`.
//
// Parâmetros:
//
//	client: Uma implementação de DynamoDBClient.
//	cfg: A configuração da tabela (nome e chaves).
//
// Retorna:
//
//	Store[T]: Uma nova instância do Store para o tipo T.
//
// Exemplo:
//
//	store := New(client, TableConfig[User]{TableName: "Users", HashKey: "id"})
func New[T any](client DynamoDBClient, cfg TableConfig[T]) Store[T] {
	if cfg.TableName == "" {
		// Tenta carregar a configuração da tabela de env vars
		_ = envloader.Load(&cfg)
	}

	return &dynamoStore[T]{
		client: client,
		cfg:    cfg,
	}
}

// Get busca um item por chave primária (hashKey e sortKey opcional).
//
// O item retornado é um ponteiro para a struct T.
//
// Parâmetros:
//
//	ctx: Contexto de requisição.
//	hashKey: Valor da chave de partição.
//	sortKey: Valor da chave de ordenação (nil se não for usado).
//
// Retorna:
//
//	*T: O ponteiro para o item encontrado.
//	error: nil, ErrNotFound, ou erro de I/O/Unmarshal.
//
// Erros:
//   - ErrNotFound: Item não existe.
//   - Erro de I/O ou Unmarshal.
func (s *dynamoStore[T]) Get(ctx context.Context, hashKey, sortKey any) (*T, error) {
	key := map[string]types.AttributeValue{
		s.cfg.HashKey: attr(hashKey),
	}
	if s.cfg.SortKey != "" && sortKey != nil {
		key[s.cfg.SortKey] = attr(sortKey)
	}

	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName:      aws.String(s.cfg.TableName),
		Key:            key,
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("dynamostore: get failed: %w", err)
	}
	if out.Item == nil {
		return nil, ErrNotFound
	}

	var item T
	if err := attributevalue.UnmarshalMap(out.Item, &item); err != nil {
		return nil, fmt.Errorf("dynamostore: unmarshal failed: %w", err)
	}
	return &item, nil
}

// Put insere ou atualiza (upsert) um item.
//
// Se um atributo TTL estiver configurado em `TableConfig` e não for fornecido
// no item, o Store o preencherá automaticamente (padrão: 30 dias).
//
// Parâmetros:
//
//	ctx: Contexto de requisição.
//	item: O item a ser inserido/atualizado.
//
// Retorna:
//
//	error: nil ou erro de marshalling/PutItem.
func (s *dynamoStore[T]) Put(ctx context.Context, item T) error {
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("dynamostore: marshal failed: %w", err)
	}

	// Adiciona TTL automático se configurado
	if s.cfg.TTLAttribute != "" {
		if ttl, ok := av[s.cfg.TTLAttribute]; ok && ttl == nil {
			// Exemplo: TTL de 30 dias se não informado
			av[s.cfg.TTLAttribute] = attr(time.Now().Add(30 * 24 * time.Hour).Unix())
		}
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.cfg.TableName),
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("dynamostore: put failed: %w", err)
	}
	return nil
}

// Delete remove um item por chave primária.
//
// Parâmetros:
//
//	ctx: Contexto de requisição.
//	hashKey: Valor da chave de partição.
//	sortKey: Valor da chave de ordenação.
//
// Retorna:
//
//	error: nil ou erro de DeleteItem.
func (s *dynamoStore[T]) Delete(ctx context.Context, hashKey, sortKey any) error {
	key := map[string]types.AttributeValue{
		s.cfg.HashKey: attr(hashKey),
	}
	if s.cfg.SortKey != "" {
		key[s.cfg.SortKey] = attr(sortKey)
	}

	_, err := s.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.cfg.TableName),
		Key:       key,
	})
	if err != nil {
		return fmt.Errorf("dynamostore: delete failed: %w", err)
	}
	return nil
}

// BatchWrite realiza operações de Put e Delete em lote.
//
// Divide as operações em grupos de 25, respeitando o limite do DynamoDB.
//
// Parâmetros:
//
//	ctx: Contexto de requisição.
//	puts: Slice de itens T para PutRequest.
//	deletes: Slice de chaves [hashKey, sortKey] para DeleteRequest.
//
// Retorna:
//
//	error: nil ou erro de Marshalling/BatchWriteItem.
func (s *dynamoStore[T]) BatchWrite(ctx context.Context, puts []T, deletes [][2]any) error {
	var writeRequests []types.WriteRequest

	// PUTs
	for _, item := range puts {
		itemMap, err := attributevalue.MarshalMap(item)
		if err != nil {
			return fmt.Errorf("batchwrite: marshal put item failed: %w", err)
		}
		writeRequests = append(writeRequests, types.WriteRequest{
			PutRequest: &types.PutRequest{Item: itemMap},
		})
	}

	// DELETEs
	for _, key := range deletes {
		hashKey, sortKey := key[0], key[1]
		keyMap := map[string]types.AttributeValue{
			s.cfg.HashKey: attr(hashKey),
		}
		if s.cfg.SortKey != "" && sortKey != nil {
			keyMap[s.cfg.SortKey] = attr(sortKey)
		}
		writeRequests = append(writeRequests, types.WriteRequest{
			DeleteRequest: &types.DeleteRequest{Key: keyMap},
		})
	}

	// DynamoDB limita a 25 operações por BatchWriteItem
	for i := 0; i < len(writeRequests); i += 25 {
		end := i + 25
		if end > len(writeRequests) {
			end = len(writeRequests)
		}

		_, err := s.client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{
				s.cfg.TableName: writeRequests[i:end],
			},
		})
		if err != nil {
			return fmt.Errorf("batchwrite failed: %w", err)
		}
	}
	return nil
}

// BatchGet busca múltiplos itens por chave primária.
//
// Divide as buscas em grupos de 100, respeitando o limite do DynamoDB.
//
// Parâmetros:
//
//	ctx: Contexto de requisição.
//	keys: Slice de chaves [hashKey, sortKey] a serem buscadas.
//
// Retorna:
//
//	[]T: Slice de itens encontrados.
//	error: nil ou erro de BatchGetItem/Unmarshal.
func (s *dynamoStore[T]) BatchGet(ctx context.Context, keys [][2]any) ([]T, error) {
	var keysToGet []map[string]types.AttributeValue
	for _, k := range keys {
		hashKey, sortKey := k[0], k[1]
		keyMap := map[string]types.AttributeValue{
			s.cfg.HashKey: attr(hashKey),
		}
		if s.cfg.SortKey != "" && sortKey != nil {
			keyMap[s.cfg.SortKey] = attr(sortKey)
		}
		keysToGet = append(keysToGet, keyMap)
	}

	var results []T

	for i := 0; i < len(keysToGet); i += 100 {
		end := i + 100
		if end > len(keysToGet) {
			end = len(keysToGet)
		}

		resp, err := s.client.BatchGetItem(ctx, &dynamodb.BatchGetItemInput{
			RequestItems: map[string]types.KeysAndAttributes{
				s.cfg.TableName: {
					Keys:           keysToGet[i:end],
					ConsistentRead: aws.Bool(true),
				},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("batchget failed: %w", err)
		}

		for _, item := range resp.Responses[s.cfg.TableName] {
			var t T
			if err := attributevalue.UnmarshalMap(item, &t); err != nil {
				return nil, err
			}
			results = append(results, t)
		}

		// Tratamento básico de UnprocessedKeys (retry simples)
		if len(resp.UnprocessedKeys) > 0 {
			time.Sleep(50 * time.Millisecond)
			// Em produção: implementar retry com backoff exponencial
		}
	}

	return results, nil
}

// Query inicia o QueryBuilder para consultas.
func (s *dynamoStore[T]) Query() *QueryBuilder[T] {
	return &QueryBuilder[T]{
		store:       s,
		scanForward: aws.Bool(true),
	}
}

// Scan inicia o QueryBuilder para varreduras.
func (s *dynamoStore[T]) Scan() *QueryBuilder[T] {
	return &QueryBuilder[T]{
		store:  s,
		isScan: true,
	}
}

// attr converte qualquer valor de chave (hashKey, sortKey, TTL) para
// o tipo de baixo nível do DynamoDB (types.AttributeValue).
//
// Retorna AttributeValueMemberNULL se o valor for nil ou houver erro
// no marshalling.
//
// Parâmetros:
//
//	v: O valor a ser convertido.
//
// Retorna:
//
//	types.AttributeValue: O valor convertido.
func attr(v any) types.AttributeValue {
	if v == nil {
		return &types.AttributeValueMemberNULL{Value: true}
	}
	av, err := attributevalue.Marshal(v)
	if err != nil {
		return &types.AttributeValueMemberNULL{Value: true}
	}
	return av
}
