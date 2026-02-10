package models

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func TestTestItem_Validation(t *testing.T) {
	validate := validator.New()

	t.Run("Deve validar com sucesso quando todos campos preenchidos", func(t *testing.T) {
		item := TestItem{
			ID:   "123",
			Name: "Produto Teste",
		}

		err := validate.Struct(item)
		assert.NoError(t, err)
	})

	t.Run("Deve falhar quando ID estiver vazio", func(t *testing.T) {
		item := TestItem{
			ID:   "", // Required falha aqui
			Name: "Produto Sem ID",
		}

		err := validate.Struct(item)
		assert.Error(t, err)

		// Verifica se o erro é realmente no campo ID
		validationErrors := err.(validator.ValidationErrors)
		assert.Equal(t, "ID", validationErrors[0].Field())
		assert.Equal(t, "required", validationErrors[0].Tag())
	})

	t.Run("Deve falhar quando Name estiver vazio", func(t *testing.T) {
		item := TestItem{
			ID:   "123",
			Name: "", // Required falha aqui
		}

		err := validate.Struct(item)
		assert.Error(t, err)

		validationErrors := err.(validator.ValidationErrors)
		assert.Equal(t, "Name", validationErrors[0].Field())
	})
}
