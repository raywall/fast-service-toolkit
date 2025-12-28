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

// EasyRepository gerencia a comunicação direta com o driver do DynamoDB (dyndb).
// Seus métodos são internos ao pacote, incentivando o uso através do EasyService.
type EasyRepository[T any] struct {
	config dyndb.TableConfig[T]
	store  dyndb.Store[T]
}

// NewRepository inicializa o armazenamento para o tipo genérico T.
func NewRepository[T any](client *dynamodb.Client, tableConfig dyndb.TableConfig[T]) *EasyRepository[T] {
	return &EasyRepository[T]{
		config: tableConfig,
		store:  dyndb.New(client, tableConfig),
	}
}

// list executa um Scan na tabela DynamoDB.
func (r *EasyRepository[T]) list(ctx context.Context) ([]T, string, error) {
	return r.store.Scan().Exec(ctx)
}

// create utiliza a operação PutItem para persistir a struct no banco.
func (r *EasyRepository[T]) create(ctx context.Context, user *T) error {
	return r.store.Put(ctx, *user)
}

// get busca um item específico. Se sk for vazio na configuração, realiza a busca apenas pela pk.
func (r *EasyRepository[T]) get(ctx context.Context, pk, sk any) (*T, error) {
	if r.config.SortKey == "" {
		return r.store.Get(ctx, pk, nil)
	}
	return r.store.Get(ctx, pk, sk)
}

// update realiza a atualização do item (atualmente via PutItem/Sobrescrita).
func (r *EasyRepository[T]) update(ctx context.Context, user *T) error {
	return r.store.Put(ctx, *user)
}

// delete remove o item através das chaves informadas.
func (r *EasyRepository[T]) delete(ctx context.Context, pk, sk any) error {
	if r.config.SortKey == "" {
		return r.store.Delete(ctx, pk, nil)
	}
	return r.store.Delete(ctx, pk, sk)
}
