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
package envloader

import (
	"fmt"
	"reflect"
)

// InvalidConfigError é retornado quando a função Load recebe um argumento 'config'
// que não é um ponteiro para uma struct.
type InvalidConfigError struct {
	// Value é o tipo refletido que foi fornecido (ex: reflect.String, reflect.Ptr).
	Value reflect.Type
}

// Error retorna uma mensagem formatada indicando o tipo de argumento inválido.
//
// O método é implementado para satisfazer a interface Go `error`.
//
// Exemplo de Retorno: "envloader: config must be a pointer to struct, got string"
func (e *InvalidConfigError) Error() string {
	if e.Value.Kind() != reflect.Ptr {
		return fmt.Sprintf("envloader: config must be a pointer to struct, got %s", e.Value.Kind())
	}
	return fmt.Sprintf("envloader: config must be a pointer to struct, got pointer to %s", e.Value.Elem().Kind())
}

// FieldError é retornado quando ocorre um erro ao tentar definir o valor de
// um campo específico da struct.
//
// Tipicamente encapsula um erro de conversão de tipo (`strconv`) ou
// um `UnsupportedTypeError`.
type FieldError struct {
	// FieldName é o nome do campo da struct (ex: "Port").
	FieldName string
	// EnvVar é o nome da variável de ambiente (ex: "APP_PORT").
	EnvVar string
	// Value é o valor bruto da variável de ambiente que causou o erro (ex: "abc").
	Value string
	// Err é o erro original encapsulado (ex: *strconv.NumError).
	Err error
}

// Error retorna uma mensagem detalhada do erro de campo.
func (e *FieldError) Error() string {
	return fmt.Sprintf("envloader: error setting field %s from env %s=%s: %v",
		e.FieldName, e.EnvVar, e.Value, e.Err)
}

// Unwrap retorna o erro original que causou o FieldError,
// implementando a interface `Unwrap` para Go 1.13+.
func (e *FieldError) Unwrap() error {
	return e.Err
}

// UnsupportedTypeError é retornado quando o tipo do campo da struct
// (ex: map, slice, interface) não é suportado pelo `envloader` para conversão.
type UnsupportedTypeError struct {
	// Type é o tipo refletido do campo não suportado.
	Type reflect.Type
}

// Error retorna uma mensagem indicando o tipo que não possui suporte.
func (e *UnsupportedTypeError) Error() string {
	return fmt.Sprintf("envloader: unsupported type %s", e.Type)
}
