package envloader

import (
	"fmt"
	"reflect"
)

// InvalidConfigError é retornado quando o config não é um ponteiro para struct
type InvalidConfigError struct {
	Value reflect.Type
}

func (e *InvalidConfigError) Error() string {
	if e.Value.Kind() != reflect.Ptr {
		return fmt.Sprintf("envloader: config must be a pointer to struct, got %s", e.Value.Kind())
	}
	return fmt.Sprintf("envloader: config must be a pointer to struct, got pointer to %s", e.Value.Elem().Kind())
}

// FieldError é retornado quando há erro ao definir um campo específico
type FieldError struct {
	FieldName string
	EnvVar    string
	Value     string
	Err       error
}

func (e *FieldError) Error() string {
	return fmt.Sprintf("envloader: error setting field %s from env %s=%s: %v",
		e.FieldName, e.EnvVar, e.Value, e.Err)
}

func (e *FieldError) Unwrap() error {
	return e.Err
}

// UnsupportedTypeError é retornado quando o tipo do campo não é suportado
type UnsupportedTypeError struct {
	Type reflect.Type
}

func (e *UnsupportedTypeError) Error() string {
	return fmt.Sprintf("envloader: unsupported type %s", e.Type)
}
