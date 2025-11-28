package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUser_StructTags(t *testing.T) {
	user := User{
		UserID:    "123",
		Email:     "john@example.com",
		Name:      "John Doe",
		Status:    "active",
		CreatedAt: 1609459200,
		ExpiresAt: 1893456000,
	}

	// Testa apenas a estrutura existente - sem métodos adicionais
	assert.Equal(t, "123", user.UserID)
	assert.Equal(t, "john@example.com", user.Email)
	assert.Equal(t, "John Doe", user.Name)
	assert.Equal(t, "active", user.Status)
	assert.Equal(t, int64(1609459200), user.CreatedAt)
	assert.Equal(t, int64(1893456000), user.ExpiresAt)
}

func TestUser_DefaultValues(t *testing.T) {
	user := User{}

	// Testa valores zero
	assert.Equal(t, "", user.UserID)
	assert.Equal(t, "", user.Email)
	assert.Equal(t, "", user.Name)
	assert.Equal(t, "", user.Status)
	assert.Equal(t, int64(0), user.CreatedAt)
	assert.Equal(t, int64(0), user.ExpiresAt)
}

func TestUser_WithTTL(t *testing.T) {
	user := User{
		UserID:    "123",
		ExpiresAt: 1893456000, // Tem TTL definido
	}

	assert.Equal(t, int64(1893456000), user.ExpiresAt)
}

func TestUser_WithoutTTL(t *testing.T) {
	user := User{
		UserID: "123",
		// ExpiresAt é 0 (zero value) - sem TTL
	}

	assert.Equal(t, int64(0), user.ExpiresAt)
}
