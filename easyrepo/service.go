package easyrepo

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/go-playground/validator/v10"
	"github.com/raywall/fast-service-toolkit/dyndb"
)

type HookType int

const (
	BeforeCreate HookType = iota
	BeforeUpdate
)

var (
	ErrEmptyCustomMethodName = errors.New("empty custom service method name")
	ErrMethodNameNotFound = errors.New("method name not found")

)

// EasyService centralizes business logic and data validation
// It encapsulates the repository and uses the validator to ensure data integrity
type EasyService[T any] struct {
	valid                *validator.Validate
	repo                 *EasyRepository[T]
	customServiceMethods map[string]CustomServiceMethod[T]
	hooks                *Hooks[T]
}

// Hooks stores the data validations and business logic registered for 
// execution before creates and updates
type Hooks[T any] struct {
	BeforeCreate []BeforeSaveHook[T]
	BeforeUpdate []BeforeSaveHook[T]
}

// BeforeSaveHook allows you to create custom validation and/or transformation functions
// which are applied before performing the update or create, allowing code injection
// customized in the easyrepo library
type BeforeSaveHook[T any] func(ctx context.Context, item *T, existing *T) error

// CustomServiceMethod allows you to inject a custom method
type CustomServiceMethod[T any] func(ctx context.Context, args ...any) (*T, error)

// NewService creates a new EasyService instance with a default validator and configured repository
func NewService[T any](client *dynamodb.Client, tableConfig dyndb.TableConfig[T]) (*EasyService[T], error) {
	return &EasyService[T]{
		valid:                validator.New(),
		repo:                 NewRepository(client, tableConfig),
		customServiceMethods: make(map[string]CustomServiceMethod[T]),
		hooks: &Hooks[T]{
			BeforeCreate: make([]BeforeSaveHook[T], 0),
			BeforeUpdate: make([]BeforeSaveHook[T], 0),
		},
	}, nil
}

// RegisterHook allows the injection of custom logic for validating and handling the request
func (s *EasyService[T]) RegisterHook(hookType HookType, fn BeforeSaveHook[T]) {
	switch hookType {
	case BeforeCreate:
		beforeCreate := s.hooks.BeforeCreate
		s.hooks.BeforeCreate = append(beforeCreate, fn)
	case BeforeUpdate:
		beforeUpdate := s.hooks.BeforeUpdate
		s.hooks.BeforeUpdate = append(beforeUpdate, fn)
	default:
		return
	}
}

// RegisterCustomServiceMethod allows you to inject a custom method
func (s *EasyService[T]) RegisterCustomServiceMethod(name string, fn CustomServiceMethod[T]) {
	s.customServiceMethods[name] = fn
}

// RegisterValidation allows adding custom validation rules to validator
func (s *EasyService[T]) RegisterValidation(name string, fn validator.Func) error {
	if s.valid == nil {
		s.valid = validator.New()
	}
	if err := s.valid.RegisterValidation(name, fn); err != nil {
		return err
	}
	return nil
}

// Get retrieves an item through its HashKey (pk) and SortKey (sk)
// Returns ErrInvalidInput if required keys are null
func (s *EasyService[T]) Get(ctx context.Context, pk, sk any) (*T, error) {
	if s.repo.Config.HashKey != "" && pk == nil {
		return nil, ErrInvalidInput
	}
	if s.repo.Config.SortKey != "" && sk == nil {
		return nil, ErrInvalidInput
	}
	return s.repo.get(ctx, pk, sk)
}

// List returns all table items (Scan operation) and the cursor for pagination
func (s *EasyService[T]) List(ctx context.Context) ([]T, string, error) {
	return s.repo.list(ctx)
}

// Create validates the struct according to the `validate` tags and persists the item in the database
func (s *EasyService[T]) Create(ctx context.Context, item *T) error {
	if err := s.valid.StructCtx(ctx, *item); err != nil {
		return err
	}
	for _, hook := range s.hooks.BeforeCreate {
		if err := hook(ctx, item, nil); err != nil {
			return err
		}
	}
	if err := s.repo.create(ctx, item); err != nil {
		return err
	}
	return nil
}

// Update validates the item and updates the data in DynamoDB
// Note: Currently requires that the repository be able to identify the item for overwriting
func (s *EasyService[T]) Update(ctx context.Context, item *T) error {
	if err := s.valid.StructCtx(ctx, *item); err != nil {
		return err
	}

	existing, err := s.repo.get(ctx, nil, nil)
	if err != nil {
		return err
	}
	for _, hook := range s.hooks.BeforeUpdate {
		if err := hook(ctx, item, existing); err != nil {
			return err
		}
	}
	if err := s.repo.update(ctx, existing); err != nil {
		return err
	}
	return nil
}

// Delete removes an item based on its primary keys
func (s *EasyService[T]) Delete(ctx context.Context, pk, sk any) error {
	if s.repo.Config.HashKey != "" && pk == nil {
		return ErrInvalidInput
	}
	if s.repo.Config.SortKey != "" && sk == nil {
		return ErrInvalidInput
	}
	return s.repo.delete(ctx, pk, sk)
}

// RunCustomServiceMethod allows you to execute a custom service method
func (s *EasyService[T]) RunCustomServiceMethod(name string, args ...any) (*T, error) {
	if name == "" {
		return nil, ErrEmptyCustomMethodName
	}
	if _, ok := s.customServiceMethods[name]; !ok {
		return nil, ErrMethodNameNotFound
	}
	if fn, ok := s.customServiceMethods[name]; ok {
		return fn(context.Background(), args...)
	}
	return nil, nil
}