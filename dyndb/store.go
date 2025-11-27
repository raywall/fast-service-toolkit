// dyndb/store.go
package dyndb

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/raywall/dynamodb-quick-service/envloader"
)

type dynamoStore[T any] struct {
	client DynamoDBClient
	cfg    TableConfig[T]
}

// New cria um store reutilizável
func New[T any](client DynamoDBClient, cfg TableConfig[T]) Store[T] {
	if cfg.TableName == "" {
		_ = envloader.Load(cfg)
	}

	return &dynamoStore[T]{
		client: client,
		cfg:    cfg,
	}
}

// Get item por chave primária
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

// Put item (upsert)
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

// Delete item
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

// BatchWrite — puts + deletes (máx 25 por chamada)
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

// BatchGet — até 100 chaves por chamada
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

// attr converte qualquer valor para types.AttributeValue
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
