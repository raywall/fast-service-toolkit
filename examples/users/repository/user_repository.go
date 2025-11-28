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
package repository

import (
	"context"
	"time"

	"github.com/raywall/fast-service-toolkit/dyndb"
	"github.com/raywall/fast-service-toolkit/examples/users/models"
)

type UserRepository struct {
	store dyndb.Store[models.User]
}

func NewUserRepository(client dyndb.DynamoDBClient) *UserRepository {
	return &UserRepository{
		store: dyndb.New(client, dyndb.TableConfig[models.User]{
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
