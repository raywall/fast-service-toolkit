package models

type TestItem struct {
	ID   string `validate:"required"`
	Name string `validate:"required"`
}
