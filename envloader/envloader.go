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
	"os"
	"reflect"
	"strconv"
	"strings"
)

// Load preenche uma struct com valores de variáveis de ambiente.
//
// A função itera sobre os campos da struct e usa as tags "env" para buscar
// o valor correspondente. Se a variável de ambiente não existir,
// o valor de "envDefault" será usado.
//
// Parâmetros:
//
//	config: Um ponteiro para a struct que será preenchida. Deve ser um ponteiro para struct.
//
// Retorna:
//
//	error: Retorna nil em caso de sucesso ou um erro tipado (`InvalidConfigError`,
//	  `FieldError`) se a operação falhar.
//
// Exemplo:
//
//	cfg := &Config{}
//	err := Load(cfg)
//
// Erros:
//   - InvalidConfigError: Se 'config' não for um ponteiro para struct.
//   - FieldError: Se houver falha na conversão de tipo de uma variável de ambiente para o tipo do campo.
func Load(config interface{}) error {
	val := reflect.ValueOf(config)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return &InvalidConfigError{Value: val.Type()}
	}

	return loadStruct(val.Elem())
}

// loadStruct processa recursivamente uma struct (ou struct aninhada).
//
// Este método é o núcleo da lógica de reflection.
func loadStruct(val reflect.Value) error {
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Verifica se o campo é exportado
		if !field.CanSet() {
			continue
		}

		// Se o campo é uma struct (aninhada), processa recursivamente
		if field.Kind() == reflect.Struct {
			if err := loadStruct(field); err != nil {
				return err
			}
			continue
		}

		// Se é um ponteiro para struct, cria a struct e processa
		if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Struct {
			if field.IsNil() {
				field.Set(reflect.New(field.Type().Elem()))
			}
			if err := loadStruct(field.Elem()); err != nil {
				return err
			}
			continue
		}

		// Obtém as tags apenas para campos não-struct
		envTag := fieldType.Tag.Get("env")
		defaultTag := fieldType.Tag.Get("default")

		// Se não tem tag env, ignora o campo
		if envTag == "" {
			continue
		}

		// Tenta obter o valor da variável de ambiente
		envValue := os.Getenv(envTag)

		// Se não encontrou, usa o valor default
		if envValue == "" {
			envValue = defaultTag
		}

		// Se ainda está vazio, continua sem alterar o campo
		if envValue == "" {
			continue
		}

		// Converte e define o valor baseado no tipo do campo
		if err := setFieldValue(field, envValue); err != nil {
			return &FieldError{
				FieldName: fieldType.Name,
				EnvVar:    envTag,
				Value:     envValue,
				Err:       err,
			}
		}
	}

	return nil
}

// setFieldValue define o valor de um campo de reflection (reflect.Value)
// baseado no seu tipo (string, int, bool, float).
//
// O valor de entrada deve ser uma string, que é convertida para o tipo
// nativo do campo Go.
//
// Parâmetros:
//
//	field: O campo Go (reflect.Value) a ser modificado.
//	value: A string contendo o valor da variável de ambiente.
//
// Retorna:
//
//	error: Retorna nil em caso de sucesso ou um erro se a conversão falhar,
//	  ou se o tipo do campo não for suportado (`UnsupportedTypeError`).
//
// Erros:
//   - UnsupportedTypeError: Se o `field.Kind()` não estiver listado no switch.
//   - Erros de conversão do `strconv` (ex: "abc" para int).
func setFieldValue(field reflect.Value, value string) error {
	if !field.CanSet() {
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(intValue)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintValue, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(uintValue)

	case reflect.Bool:
		boolValue, err := strconv.ParseBool(strings.ToLower(value))
		if err != nil {
			return err
		}
		field.SetBool(boolValue)

	case reflect.Float32, reflect.Float64:
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.SetFloat(floatValue)

	default:
		return &UnsupportedTypeError{Type: field.Type()}
	}

	return nil
}

// MustLoad é similar ao Load, mas provoca um panic em caso de erro.
//
// Deve ser usado para configurações essenciais onde a falha na inicialização
// do programa é inaceitável.
//
// Parâmetros:
//
//	config: Um ponteiro para a struct de configuração.
//
// Exemplo:
//
//	cfg := &Config{}
//	MustLoad(cfg) // O programa termina se houver erro
func MustLoad(config interface{}) {
	if err := Load(config); err != nil {
		panic(err)
	}
}
