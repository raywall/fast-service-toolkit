package envloader

import (
	"os"
	"reflect"
	"strconv"
	"strings"
)

// Load preenche uma struct com valores de variáveis de ambiente
// baseado nas tags "env" e "envDefault"
func Load(config interface{}) error {
	val := reflect.ValueOf(config)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return &InvalidConfigError{Value: val.Type()}
	}

	return loadStruct(val.Elem())
}

// loadStruct processa recursivamente uma struct
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
		defaultTag := fieldType.Tag.Get("envDefault")

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

// setFieldValue define o valor de um campo baseado no seu tipo
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

// MustLoad é similar ao Load, mas panic em caso de erro
func MustLoad(config interface{}) {
	if err := Load(config); err != nil {
		panic(err)
	}
}