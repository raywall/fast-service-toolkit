package easyrepo

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/go-playground/validator/v10"
	"github.com/raywall/fast-service-toolkit/dyndb"
)

// EasyService centraliza a lógica de negócio e validação de dados.
// Ele encapsula o repositório e utiliza o validador para garantir a integridade dos dados.
type EasyService[T any] struct {
	valid *validator.Validate
	repo  *EasyRepository[T]
}

// NewService cria uma nova instância de EasyService com um validador padrão e o repositório configurado.
func NewService[T any](client *dynamodb.Client, tableConfig dyndb.TableConfig[T]) (*EasyService[T], error) {
	return &EasyService[T]{
		valid: validator.New(),
		repo:  NewRepository(client, tableConfig),
	}, nil
}

// RegisterValidation permite adicionar regras de validação personalizadas ao validator/v10.
func (s *EasyService[T]) RegisterValidation(name string, fn validator.Func) error {
	if s.valid == nil {
		s.valid = validator.New()
	}
	if err := s.valid.RegisterValidation(name, fn); err != nil {
		return err
	}
	return nil
}

// Get recupera um item através de sua HashKey (pk) e SortKey (sk).
// Retorna ErrInvalidInput se as chaves obrigatórias forem nulas.
func (s *EasyService[T]) Get(ctx context.Context, pk, sk any) (*T, error) {
	if s.repo.config.HashKey != "" && pk == nil {
		return nil, ErrInvalidInput
	}
	if s.repo.config.SortKey != "" && sk == nil {
		return nil, ErrInvalidInput
	}
	return s.repo.get(ctx, pk, sk)
}

// List retorna todos os itens da tabela (operação de Scan) e o cursor para paginação.
func (s *EasyService[T]) List(ctx context.Context) ([]T, string, error) {
	return s.repo.list(ctx)
}

// Create valida a struct conforme as tags `validate` e persiste o item no banco de dados.
func (s *EasyService[T]) Create(ctx context.Context, item *T) error {
	if err := s.valid.StructCtx(ctx, *item); err != nil {
		return err
	}
	if err := s.repo.create(ctx, item); err != nil {
		return err
	}
	return nil
}

// Update valida o item e atualiza os dados no DynamoDB.
// Nota: Atualmente requer que o repositório consiga identificar o item para sobrescrita.
func (s *EasyService[T]) Update(ctx context.Context, item *T) error {
	if err := s.valid.StructCtx(ctx, *item); err != nil {
		return err
	}

	existing, err := s.repo.get(ctx, nil, nil)
	if err != nil {
		return err
	}
	if err := s.repo.update(ctx, existing); err != nil {
		return err
	}
	return nil
}

// Delete remove um item baseado em suas chaves primárias.
func (s *EasyService[T]) Delete(ctx context.Context, pk, sk any) error {
	if s.repo.config.HashKey != "" && pk == nil {
		return ErrInvalidInput
	}
	if s.repo.config.SortKey != "" && sk == nil {
		return ErrInvalidInput
	}
	return s.repo.delete(ctx, pk, sk)
}
