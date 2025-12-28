package easyrepo

import (
	"context"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/raywall/fast-service-toolkit/dyndb"
	"github.com/raywall/fast-service-toolkit/easyrepo/models"
	"github.com/stretchr/testify/assert"
)

func TestEasyService_Create_Validation(t *testing.T) {
	config := dyndb.TableConfig[models.TestItem]{HashKey: "ID"}
	service, _ := NewService(nil, config)

	t.Run("should return error when item is invalid", func(ctx *testing.T) {
		invalidItem := &models.TestItem{ID: ""} // Name e ID vazios (required)
		err := service.Create(context.Background(), invalidItem)

		assert.Error(t, err)
	})

	t.Run("should fail validation for custom rule", func(t *testing.T) {
		_ = service.RegisterValidation("is-admin", func(fl validator.FieldLevel) bool {
			return fl.Field().String() == "admin"
		})
	})
}

func TestEasyService_Get_InputCheck(t *testing.T) {
	config := dyndb.TableConfig[models.TestItem]{HashKey: "PK", SortKey: "SK"}
	service, _ := NewService(nil, config)

	t.Run("should return ErrInvalidInput when PK is missing", func(t *testing.T) {
		_, err := service.Get(context.Background(), nil, "some-sk")
		assert.Equal(t, ErrInvalidInput, err)
	})

	t.Run("should return ErrInvalidInput when SK is missing", func(t *testing.T) {
		_, err := service.Get(context.Background(), "some-pk", nil)
		assert.Equal(t, ErrInvalidInput, err)
	})
}
