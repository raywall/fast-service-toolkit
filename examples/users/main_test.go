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
package main

// import (
// 	"context"
// 	"testing"

// 	"github.com/raywall/fast-service-toolkit/dyndb"
// 	"github.com/raywall/fast-service-toolkit/examples/users/models"
// 	"github.com/raywall/fast-service-toolkit/examples/users/repository"

// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// )

// func TestMain_ExampleUsage(t *testing.T) {
// 	// Mock store para demonstrar o uso básico
// 	mockStore := &dyndb.MockStore[models.User]{
// 		GetFn: func(ctx context.Context, hashKey, sortKey any) (*models.User, error) {
// 			if hashKey == "user-1" {
// 				return &models.User{
// 					UserID: "user-1",
// 					Email:  "test@example.com",
// 					Name:   "Test User",
// 					Status: "active",
// 				}, nil
// 			}
// 			return nil, dyndb.ErrNotFound
// 		},
// 		PutFn: func(ctx context.Context, user models.User) error {
// 			return nil
// 		},
// 	}

// 	repo := repository.NewUserRepository(mockStore)
// 	ctx := context.Background()

// 	t.Run("create and retrieve user", func(t *testing.T) {
// 		// Create user
// 		user := models.User{
// 			UserID: "user-1",
// 			Email:  "test@example.com",
// 			Name:   "Test User",
// 			Status: "active",
// 		}

// 		err := repo.Save(ctx, &user)
// 		assert.NoError(t, err)

// 		// Get user
// 		retrievedUser, err := repo.GetByID(ctx, "user-1")
// 		require.NoError(t, err)
// 		assert.Equal(t, "Test User", retrievedUser.Name)
// 		assert.Equal(t, "test@example.com", retrievedUser.Email)
// 	})

// 	t.Run("user not found", func(t *testing.T) {
// 		user, err := repo.GetByID(ctx, "non-existent")
// 		assert.ErrorIs(t, err, dyndb.ErrNotFound)
// 		assert.Nil(t, user)
// 	})
// }

// func TestMain_BatchOperations(t *testing.T) {
// 	mockStore := &dyndb.MockStore[models.User]{
// 		BatchGetFn: func(ctx context.Context, keys [][2]any) ([]models.User, error) {
// 			users := make([]models.User, 0)
// 			for _, key := range keys {
// 				if key[0] == "user-1" {
// 					users = append(users, models.User{
// 						UserID: "user-1",
// 						Email:  "user1@example.com",
// 						Name:   "User One",
// 					})
// 				}
// 			}
// 			return users, nil
// 		},
// 	}

// 	// Demonstração do uso direto do mock store para batch operations
// 	users, err := mockStore.BatchGet(context.Background(), [][2]any{
// 		{"user-1", nil},
// 		{"user-2", nil},
// 	})

// 	require.NoError(t, err)
// 	assert.Len(t, users, 1)
// 	assert.Equal(t, "user-1", users[0].UserID)
// }

// func TestMain_QueryOperations(t *testing.T) {
// 	mockStore := &dyndb.MockStore[models.User]{
// 		QueryFn: func() *dyndb.MockQueryBuilder[models.User] {
// 			return &dyndb.MockQueryBuilder[models.User]{
// 				ExecFn: func(ctx context.Context) ([]models.User, string, error) {
// 					return []models.User{
// 						{
// 							UserID: "query-user",
// 							Email:  "query@example.com",
// 							Name:   "Query User",
// 							Status: "active",
// 						},
// 					}, "next-token", nil
// 				},
// 			}
// 		},
// 	}

// 	repo := repository.NewUserRepository(mockStore)
// 	users, _, err := repo.ListActive(context.Background(), 10, "")

// 	require.NoError(t, err)
// 	require.Len(t, users, 1)
// 	assert.Equal(t, "query-user", users[0].UserID)
// }

// func TestMain_ScanOperations(t *testing.T) {
// 	mockStore := &dyndb.MockStore[models.User]{
// 		ScanFn: func() *dyndb.MockQueryBuilder[models.User] {
// 			return &dyndb.MockQueryBuilder[models.User]{
// 				ExecFn: func(ctx context.Context) ([]models.User, string, error) {
// 					return []models.User{
// 						{
// 							UserID: "scan-user-1",
// 							Email:  "scan1@example.com",
// 							Name:   "Scan User 1",
// 						},
// 						{
// 							UserID: "scan-user-2",
// 							Email:  "scan2@example.com",
// 							Name:   "Scan User 2",
// 						},
// 					}, "", nil
// 				},
// 			}
// 		},
// 	}

// 	repo := repository.NewUserRepository(mockStore)
// 	users, err := repo.BatchGetByIDs(context.Background(), []string{"scan-user-1", "scan-user-2"})

// 	require.NoError(t, err)
// 	assert.Len(t, users, 2)
// }
