package easyrepo

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/raywall/fast-service-toolkit/dyndb"
)

var (
	ErrUserNotFound      = errors.New("item not found")
	ErrInvalidInput      = errors.New("invalid input")
	ErrUserAlreadyExists = errors.New("item already exists")
)

// EasyRepository manages direct communication with the DynamoDB driver (dyndb)
// Its methods are internal to the package, encouraging use through EasyService
type EasyRepository[T any] struct {
	Config dyndb.TableConfig[T]
	Store  dyndb.Store[T]
}

// NewRepository initializes storage for generic type T
func NewRepository[T any](client *dynamodb.Client, tableConfig dyndb.TableConfig[T]) *EasyRepository[T] {
	return &EasyRepository[T]{
		Config: tableConfig,
		Store:  dyndb.New(client, tableConfig),
	}
}

// list performs a Scan on the DynamoDB table
func (r *EasyRepository[T]) list(ctx context.Context) ([]T, string, error) {
	return r.Store.Scan().Exec(ctx)
}

// create uses the PutItem operation to persist the struct in the database
func (r *EasyRepository[T]) create(ctx context.Context, user *T) error {
	return r.Store.Put(ctx, *user)
}

// get Search for a specific item. If sk is empty in the configuration, it searches only for pk
func (r *EasyRepository[T]) get(ctx context.Context, pk, sk any) (*T, error) {
	if r.Config.SortKey == "" {
		return r.Store.Get(ctx, pk, nil)
	}
	return r.Store.Get(ctx, pk, sk)
}

// update performs the item update (currently via PutItem/Subscrive)
func (r *EasyRepository[T]) update(ctx context.Context, user *T) error {
	return r.Store.Put(ctx, *user)
}

// delete removes the item using the keys provided
func (r *EasyRepository[T]) delete(ctx context.Context, pk, sk any) error {
	if r.Config.SortKey == "" {
		return r.Store.Delete(ctx, pk, nil)
	}
	return r.Store.Delete(ctx, pk, sk)
}
