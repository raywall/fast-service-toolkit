// dyndb/query.go
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

// === MÉTODOS FLUENTES (inferência automática garantida!) ===

func (qb *QueryBuilder[T]) Index(name string) *QueryBuilder[T] {
	qb.indexName = aws.String(name)
	return qb
}

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

func (qb *QueryBuilder[T]) Limit(n int32) *QueryBuilder[T] {
	qb.limit = &n
	return qb
}

func (qb *QueryBuilder[T]) LastKey(token string) *QueryBuilder[T] {
	if token == "" {
		return qb
	}
	if data, err := base64.StdEncoding.DecodeString(token); err == nil {
		_ = json.Unmarshal(data, &qb.lastKey)
	}
	return qb
}

// Query inicia uma Query
func (s *dynamoStore[T]) Query() *QueryBuilder[T] {
	return &QueryBuilder[T]{
		store:       s,
		scanForward: aws.Bool(true),
	}
}

// Scan inicia um Scan
func (s *dynamoStore[T]) Scan() *QueryBuilder[T] {
	return &QueryBuilder[T]{
		store:  s,
		isScan: true,
	}
}

// Filtros aplica filtros utilizando inferência de tipo automática
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

func WithIndex[T any](name string) QueryFilter[T] {
	return func(qb *QueryBuilder[T]) {
		qb.indexName = aws.String(name)
	}
}

func WithLimit[T any](n int32) QueryFilter[T] {
	return func(qb *QueryBuilder[T]) {
		qb.limit = &n
	}
}

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

func WithScanForward[T any](forward bool) QueryFilter[T] {
	return func(qb *QueryBuilder[T]) {
		qb.scanForward = &forward
	}
}

// Exec executa a consulta com filtros opcionais
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
