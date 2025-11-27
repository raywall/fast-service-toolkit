// examples/users/repository/user_repository.go
package repository

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/raywall/dynamodb-quick-service/dyndb"
	"github.com/raywall/dynamodb-quick-service/examples/users/models"
)

type UserRepository struct {
	store dyndb.Store[models.User]
}

func NewUserRepository(client *dynamodb.Client) *UserRepository {
	return &UserRepository{
		store: dyndb.New[models.User](client, dyndb.TableConfig[models.User]{
			TableName:    "dev-users", // ou use env var
			HashKey:      "userId",
			SortKey:      "email",
			TTLAttribute: "expiresAt",
		}),
	}
}

// Save (upsert) — com CreatedAt automático
func (r *UserRepository) Save(ctx context.Context, user *models.User) error {
	if user.CreatedAt == 0 {
		user.CreatedAt = time.Now().Unix()
	}
	return r.store.Put(ctx, *user)
}

// GetByID — busca por PK (sortKey opcional)
func (r *UserRepository) GetByID(ctx context.Context, userID string) (*models.User, error) {
	return r.store.Get(ctx, userID, nil)
}

// GetByEmail — usa GSI para busca por email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	users, _, err := r.store.Query().
		Index("email-index").     // infere T automaticamente
		KeyEqual("email", email). // infere T
		Limit(10).                // infere T
		Exec(ctx)

	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, dyndb.ErrNotFound
	}
	return &users[0], nil
}

// ListActive — scan com filtro + paginação
func (r *UserRepository) ListActive(ctx context.Context, limit int32, token string) ([]models.User, string, error) {
	return r.store.Scan().
		FilterEqual("status", "active").
		Limit(limit).
		LastKey(token).
		Exec(ctx)
}

// Delete — deleta por PK + SK
func (r *UserRepository) Delete(ctx context.Context, userID, email string) error {
	return r.store.Delete(ctx, userID, email)
}

// BatchGet — exemplo de batch para múltiplos IDs
func (r *UserRepository) BatchGetByIDs(ctx context.Context, userIDs []string) ([]models.User, error) {
	keys := make([][2]any, len(userIDs))
	for i, id := range userIDs {
		keys[i] = [2]any{id, nil} // sem sortKey
	}
	return r.store.BatchGet(ctx, keys)
}

// BatchSave — salva múltiplos de uma vez
func (r *UserRepository) BatchSave(ctx context.Context, users []models.User) error {
	var deletes [][2]any // exemplo: pode deletar outros se precisar
	return r.store.BatchWrite(ctx, users, deletes)
}
