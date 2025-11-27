package repository

// import (
// 	"context"
// 	"testing"

// 	"github.com/raywall/dynamodb-quick-service/dyndb"
// 	"github.com/raywall/dynamodb-quick-service/examples/users/models"

// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// )

// func TestNewUserRepository(t *testing.T) {
// 	mockStore := &dyndb.MockStore[models.User]{}
// 	repo := NewUserRepository(mockStore)

// 	assert.NotNil(t, repo)
// 	assert.Equal(t, mockStore, repo.store)
// }

// func TestUserRepository_Create(t *testing.T) {
// 	mockStore := &dyndb.MockStore[models.User]{
// 		PutFn: func(ctx context.Context, user models.User) error {
// 			// Verifica se o usuário tem os campos esperados
// 			assert.Equal(t, "user-123", user.UserID)
// 			assert.Equal(t, "john@example.com", user.Email)
// 			assert.Equal(t, "John Doe", user.Name)
// 			return nil
// 		},
// 	}

// 	repo := NewUserRepository(mockStore)
// 	user := models.User{
// 		UserID: "user-123",
// 		Email:  "john@example.com",
// 		Name:   "John Doe",
// 		Status: "active",
// 	}

// 	err := repo.Save(context.Background(), &user)
// 	assert.NoError(t, err)
// }

// func TestUserRepository_GetByID(t *testing.T) {
// 	mockStore := &dyndb.MockStore[models.User]{
// 		GetFn: func(ctx context.Context, hashKey, sortKey any) (*models.User, error) {
// 			assert.Equal(t, "user-123", hashKey)
// 			assert.Nil(t, sortKey)
// 			return &models.User{
// 				UserID: "user-123",
// 				Email:  "john@example.com",
// 				Name:   "John Doe",
// 			}, nil
// 		},
// 	}

// 	repo := NewUserRepository(mockStore)
// 	user, err := repo.GetByID(context.Background(), "user-123")

// 	require.NoError(t, err)
// 	require.NotNil(t, user)
// 	assert.Equal(t, "user-123", user.UserID)
// 	assert.Equal(t, "john@example.com", user.Email)
// }

// func TestUserRepository_GetByID_NotFound(t *testing.T) {
// 	mockStore := &dyndb.MockStore[models.User]{
// 		GetFn: func(ctx context.Context, hashKey, sortKey any) (*models.User, error) {
// 			return nil, dyndb.ErrNotFound
// 		},
// 	}

// 	repo := NewUserRepository(mockStore)
// 	user, err := repo.GetByID(context.Background(), "non-existent")

// 	assert.ErrorIs(t, err, dyndb.ErrNotFound)
// 	assert.Nil(t, user)
// }

// func TestUserRepository_Update(t *testing.T) {
// 	mockStore := &dyndb.MockStore[models.User]{
// 		PutFn: func(ctx context.Context, user models.User) error {
// 			assert.Equal(t, "user-123", user.UserID)
// 			assert.Equal(t, "john.updated@example.com", user.Email)
// 			return nil
// 		},
// 	}

// 	repo := NewUserRepository(mockStore)
// 	user := models.User{
// 		UserID: "user-123",
// 		Email:  "john.updated@example.com",
// 		Name:   "John Updated",
// 	}

// 	err := repo.Save(context.Background(), &user)
// 	assert.NoError(t, err)
// }

// func TestUserRepository_Delete(t *testing.T) {
// 	mockStore := &dyndb.MockStore[models.User]{
// 		DeleteFn: func(ctx context.Context, hashKey, sortKey any) error {
// 			assert.Equal(t, "user-123", hashKey)
// 			assert.Nil(t, sortKey)
// 			return nil
// 		},
// 	}

// 	repo := NewUserRepository(mockStore)
// 	err := repo.Delete(context.Background(), "user-123", "john@example.com")

// 	assert.NoError(t, err)
// }

// func TestUserRepository_List(t *testing.T) {
// 	mockStore := &dyndb.MockStore[models.User]{
// 		ScanFn: func() *dyndb.MockQueryBuilder[models.User] {
// 			return &dyndb.MockQueryBuilder[models.User]{
// 				ExecFn: func(ctx context.Context) ([]models.User, string, error) {
// 					return []models.User{
// 						{
// 							UserID: "user-1",
// 							Email:  "user1@example.com",
// 							Name:   "User One",
// 						},
// 						{
// 							UserID: "user-2",
// 							Email:  "user2@example.com",
// 							Name:   "User Two",
// 						},
// 					}, "", nil
// 				},
// 			}
// 		},
// 	}

// 	repo := NewUserRepository(mockStore)
// 	users, err := repo.BatchGetByIDs(context.Background(), []string{"user-1", "user-2"})

// 	require.NoError(t, err)
// 	require.Len(t, users, 2)
// 	assert.Equal(t, "user-1", users[0].UserID)
// 	assert.Equal(t, "user-2", users[1].UserID)
// }

// func TestUserRepository_ListActive(t *testing.T) {
// 	mockStore := &dyndb.MockStore[models.User]{
// 		QueryFn: func() *dyndb.MockQueryBuilder[models.User] {
// 			return &dyndb.MockQueryBuilder[models.User]{
// 				ExecFn: func(ctx context.Context) ([]models.User, string, error) {
// 					return []models.User{
// 						{
// 							UserID: "active-user",
// 							Email:  "active@example.com",
// 							Name:   "Active User",
// 							Status: "active",
// 						},
// 					}, "", nil
// 				},
// 			}
// 		},
// 	}

// 	repo := NewUserRepository(mockStore)
// 	users, _, err := repo.ListActive(context.Background(), 10, "")

// 	require.NoError(t, err)
// 	require.Len(t, users, 1)
// 	assert.Equal(t, "active-user", users[0].UserID)
// 	assert.Equal(t, "active", users[0].Status)
// }
