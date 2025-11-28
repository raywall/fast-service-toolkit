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
	"encoding/base64"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// === MÉTODOS FLUENTES (QueryBuilder) ===

// Index define o nome do Índice Secundário Global (GSI) ou Local (LSI)
// a ser usado na consulta.
func (qb *QueryBuilder[T]) Index(name string) *QueryBuilder[T] {
	qb.indexName = aws.String(name)
	return qb
}

// KeyEqual adiciona uma condição de chave de igualdade (`KEY = VALUE`) ao KeyConditionExpression.
//
// Usado para HashKey em Query ou para a chave de ordenação (SortKey).
func (qb *QueryBuilder[T]) KeyEqual(key string, value any) *QueryBuilder[T] {
	cond := expression.KeyEqual(expression.Key(key), expression.Value(value))
	if qb.keyCond == nil {
		qb.keyCond = &cond
	} else {
		tmp := qb.keyCond.And(cond)
		qb.keyCond = &tmp
	}
	return qb
}

// KeyBeginsWith adiciona uma condição de chave "começa com" (`KEY BEGINS WITH PREFIX`)
// ao KeyConditionExpression.
//
// Usado tipicamente para SortKeys.
func (qb *QueryBuilder[T]) KeyBeginsWith(key, prefix string) *QueryBuilder[T] {
	cond := expression.Key(key).BeginsWith(prefix)
	if qb.keyCond == nil {
		qb.keyCond = &cond
	} else {
		tmp := qb.keyCond.And(cond)
		qb.keyCond = &tmp
	}
	return qb
}

// FilterEqual adiciona uma condição de filtro de igualdade (`FIELD = VALUE`) ao FilterExpression.
//
// O filtro é aplicado após a consulta/varredura (pós-processamento) e pode reduzir o Throughput.
func (qb *QueryBuilder[T]) FilterEqual(field string, value any) *QueryBuilder[T] {
	cond := expression.Equal(expression.Name(field), expression.Value(value))
	if qb.filterCond == nil {
		qb.filterCond = &cond
	} else {
		tmp := qb.filterCond.And(cond)
		qb.filterCond = &tmp
	}
	return qb
}

// FilterContains adiciona uma condição de filtro "contém" (`FIELD CONTAINS VALUE`) ao FilterExpression.
func (qb *QueryBuilder[T]) FilterContains(field string, value any) *QueryBuilder[T] {
	cond := expression.Contains(expression.Name(field), value)
	if qb.filterCond == nil {
		qb.filterCond = &cond
	} else {
		tmp := qb.filterCond.And(cond)
		qb.filterCond = &tmp
	}
	return qb
}

// Limit define o número máximo de itens a serem lidos pelo DynamoDB.
//
// Nota: O filtro é aplicado após a leitura, o número final de itens
// retornados pode ser menor que o limite.
func (qb *QueryBuilder[T]) Limit(n int32) *QueryBuilder[T] {
	qb.limit = &n
	return qb
}

// LastKey decodifica um token de paginação Base64 (retornado por Exec)
// e o utiliza como ExclusiveStartKey na próxima consulta.
func (qb *QueryBuilder[T]) LastKey(token string) *QueryBuilder[T] {
	if token == "" {
		return qb
	}
	if data, err := base64.StdEncoding.DecodeString(token); err == nil {
		_ = json.Unmarshal(data, &qb.lastKey)
	}
	return qb
}

// === FUNÇÕES DE FILTRO (QueryFilter) ===

// Funções `With...` implementam `QueryFilter[T]` e permitem aplicar filtros
// como argumentos variádicos no método `Exec`.

// WithKeyCondition permite fornecer um KeyConditionBuilder personalizado
// do SDK da AWS.
func WithKeyCondition[T any](cond expression.KeyConditionBuilder) QueryFilter[T] {
	return func(qb *QueryBuilder[T]) {
		if qb.keyCond == nil {
			qb.keyCond = &cond
		} else {
			tmp := qb.keyCond.And(cond)
			qb.keyCond = &tmp
		}
	}
}

// WithFilter permite fornecer um ConditionBuilder de filtro personalizado.
func WithFilter[T any](cond expression.ConditionBuilder) QueryFilter[T] {
	return func(qb *QueryBuilder[T]) {
		if qb.filterCond == nil {
			qb.filterCond = &cond
		} else {
			tmp := qb.filterCond.And(cond)
			qb.filterCond = &tmp
		}
	}
}

// WithIndex define o nome do índice a ser usado.
func WithIndex[T any](name string) QueryFilter[T] {
	return func(qb *QueryBuilder[T]) {
		qb.indexName = aws.String(name)
	}
}

// WithLimit define o limite de itens a serem lidos.
func WithLimit[T any](n int32) QueryFilter[T] {
	return func(qb *QueryBuilder[T]) {
		qb.limit = &n
	}
}

// WithLastEvaluatedKey usa o token de paginação Base64.
func WithLastEvaluatedKey[T any](token string) QueryFilter[T] {
	return func(qb *QueryBuilder[T]) {
		if token == "" {
			return
		}
		if data, err := base64.StdEncoding.DecodeString(token); err == nil {
			_ = json.Unmarshal(data, &qb.lastKey)
		}
	}
}

// WithScanForward define a direção da ordenação da consulta (true = ascendente).
func WithScanForward[T any](forward bool) QueryFilter[T] {
	return func(qb *QueryBuilder[T]) {
		qb.scanForward = &forward
	}
}

// Exec executa a consulta Query ou Scan construída, aplicando filtros opcionais.
//
// A decisão entre Query e Scan é baseada no estado interno do builder (`isScan`)
// ou se uma KeyCondition foi fornecida.
//
// Parâmetros:
//
//	ctx: Contexto de requisição.
//	filters: Funções QueryFilter opcionais a serem aplicadas antes da execução.
//
// Retorna:
//
//	[]T: Slice de itens encontrados.
//	string: Token de paginação Base64 para a próxima página (vazio se finalizado).
//	error: Erro de I/O ou de expressão.
func (qb *QueryBuilder[T]) Exec(ctx context.Context, filters ...QueryFilter[T]) ([]T, string, error) {
	// Aplica os filtros antes de construir a expressão
	for _, filter := range filters {
		filter(qb)
	}

	builder := expression.NewBuilder()

	hasConditions := false

	if qb.keyCond != nil {
		builder = builder.WithKeyCondition(*qb.keyCond)
		hasConditions = true
	}
	if qb.filterCond != nil {
		builder = builder.WithFilter(*qb.filterCond)
		hasConditions = true
	}
	if qb.projection != nil {
		builder = builder.WithProjection(*qb.projection)
		hasConditions = true
	}

	// Se não há condições, não precisa construir expression
	var expr expression.Expression
	var err error

	if hasConditions {
		expr, err = builder.Build()
		if err != nil {
			return nil, "", err
		}
	}

	if qb.isScan || qb.keyCond == nil {
		return qb.execScan(ctx, expr)
	}
	return qb.execQuery(ctx, expr)
}

// execQuery executa a operação Query no DynamoDB.
func (qb *QueryBuilder[T]) execQuery(ctx context.Context, expr expression.Expression) ([]T, string, error) {
	input := &dynamodb.QueryInput{
		TableName:         aws.String(qb.store.cfg.TableName),
		IndexName:         qb.indexName,
		Limit:             qb.limit,
		ScanIndexForward:  qb.scanForward,
		ExclusiveStartKey: qb.lastKey,
	}

	// Aplica as expressões apenas se existirem
	if expr.KeyCondition() != nil {
		input.KeyConditionExpression = expr.KeyCondition()
		input.ExpressionAttributeNames = expr.Names()
		input.ExpressionAttributeValues = expr.Values()
	}
	if expr.Filter() != nil {
		input.FilterExpression = expr.Filter()
		if input.ExpressionAttributeNames == nil {
			input.ExpressionAttributeNames = expr.Names()
		}
		if input.ExpressionAttributeValues == nil {
			input.ExpressionAttributeValues = expr.Values()
		}
	}
	if expr.Projection() != nil {
		input.ProjectionExpression = expr.Projection()
		if input.ExpressionAttributeNames == nil {
			input.ExpressionAttributeNames = expr.Names()
		}
	}

	out, err := qb.store.client.Query(ctx, input)
	if err != nil {
		return nil, "", err
	}
	return qb.unmarshalResults(out.Items, out.LastEvaluatedKey)
}

// execScan executa a operação Scan no DynamoDB.
func (qb *QueryBuilder[T]) execScan(ctx context.Context, expr expression.Expression) ([]T, string, error) {
	input := &dynamodb.ScanInput{
		TableName:         aws.String(qb.store.cfg.TableName),
		Limit:             qb.limit,
		ExclusiveStartKey: qb.lastKey,
	}

	// Aplica as expressões apenas se existirem
	if expr.Filter() != nil {
		input.FilterExpression = expr.Filter()
		input.ExpressionAttributeNames = expr.Names()
		input.ExpressionAttributeValues = expr.Values()
	}
	if expr.Projection() != nil {
		input.ProjectionExpression = expr.Projection()
		if input.ExpressionAttributeNames == nil {
			input.ExpressionAttributeNames = expr.Names()
		}
	}

	out, err := qb.store.client.Scan(ctx, input)
	if err != nil {
		return nil, "", err
	}
	return qb.unmarshalResults(out.Items, out.LastEvaluatedKey)
}

// unmarshalResults desserializa os resultados do DynamoDB e cria o token de paginação.
//
// Este é o método interno que lida com a conversão de tipos e Base64.
func (qb *QueryBuilder[T]) unmarshalResults(
	items []map[string]types.AttributeValue,
	lastKey map[string]types.AttributeValue,
) ([]T, string, error) {
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
