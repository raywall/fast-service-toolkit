package injector

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/raywall/fast-service-lab/pkg/enrichment"
)

// Regex para capturar padrões ${tipo.chave}
// Ex: ${env.API_KEY}, ${ssm./app/config}, ${secret.db_pass}
var pattern = regexp.MustCompile(`\$\{(env|ssm|secret)\.([^}]+)\}`)

type Injector struct{}

func New() *Injector {
	return &Injector{}
}

func (i *Injector) Inject(ctx context.Context, target interface{}) error {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return fmt.Errorf("target deve ser um ponteiro para struct não nulo")
	}
	return i.injectRecursive(ctx, v.Elem())
}

func (i *Injector) injectRecursive(ctx context.Context, v reflect.Value) error {
	switch v.Kind() {
	case reflect.Struct:
		t := v.Type()
		for k := 0; k < t.NumField(); k++ {
			field := t.Field(k)
			value := v.Field(k)

			// 1. Processa Tags (env:"...")
			if err := i.processStructTags(ctx, field, value); err != nil {
				return err
			}

			// 2. Processa Strings com Interpolação "${...}"
			if value.Kind() == reflect.String && value.CanSet() {
				newValue, err := i.interpolateString(ctx, value.String())
				if err != nil {
					return err
				}
				value.SetString(newValue)
			}

			// 3. Recursão
			if value.CanSet() || value.Kind() == reflect.Ptr {
				if err := i.injectRecursive(ctx, value); err != nil {
					return err
				}
			}
		}

	case reflect.Map:
		if v.Type().Key().Kind() == reflect.String {
			if !v.IsNil() {
				i.injectMap(ctx, v)
			}
		}

	case reflect.Ptr:
		if !v.IsNil() {
			return i.injectRecursive(ctx, v.Elem())
		}

	case reflect.Slice:
		for j := 0; j < v.Len(); j++ {
			if err := i.injectRecursive(ctx, v.Index(j)); err != nil {
				return err
			}
		}
	}
	return nil
}

// processStructTags mantém a lógica legado de tags
func (i *Injector) processStructTags(ctx context.Context, field reflect.StructField, value reflect.Value) error {
	if !value.CanSet() {
		return nil
	}
	if tag := field.Tag.Get("env"); tag != "" {
		if val, exists := os.LookupEnv(tag); exists {
			return setField(value, val)
		}
	}
	return nil
}

// interpolateString realiza a substituição baseada em Regex
func (i *Injector) interpolateString(ctx context.Context, input string) (string, error) {
	if !strings.Contains(input, "${") {
		return input, nil
	}

	var err error
	// ReplaceAllStringFunc permite lógica customizada para cada match
	result := pattern.ReplaceAllStringFunc(input, func(match string) string {
		// match é algo como "${env.VAR_NAME}"
		// Removemos ${ e }
		content := match[2 : len(match)-1] // env.VAR_NAME
		parts := strings.SplitN(content, ".", 2)
		if len(parts) != 2 {
			return match // Formato inválido, retorna original
		}

		sourceType := parts[0]
		key := parts[1]

		val, resolveErr := i.fetchValue(ctx, sourceType, key)
		if resolveErr != nil {
			err = resolveErr // Captura erro para retornar depois
			return match
		}

		// Converte qualquer valor para string para interpolação
		return fmt.Sprintf("%v", val)
	})

	return result, err
}

// injectMap lida com mapas dinâmicos
func (i *Injector) injectMap(ctx context.Context, v reflect.Value) {
	iter := v.MapRange()
	updates := make(map[string]interface{})

	for iter.Next() {
		key := iter.Key()
		val := iter.Value()

		elem := val
		if val.Kind() == reflect.Interface {
			elem = val.Elem()
		}

		if !elem.IsValid() {
			continue
		}

		if elem.Kind() == reflect.String {
			// Usa a nova lógica de interpolação também para mapas
			newVal, _ := i.interpolateString(ctx, elem.String())
			updates[key.String()] = newVal
		} else if elem.Kind() == reflect.Map {
			if subMap, ok := elem.Interface().(map[string]interface{}); ok {
				subVal := reflect.ValueOf(subMap)
				i.injectMap(ctx, subVal)
			}
		}
	}

	for k, val := range updates {
		v.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(val))
	}
}

// fetchValue centraliza a busca de dados
func (i *Injector) fetchValue(ctx context.Context, sourceType, key string) (interface{}, error) {
	switch sourceType {
	case "env":
		if val, exists := os.LookupEnv(key); exists {
			return val, nil
		}
		return "", nil // Variável não encontrada retorna vazio (ou erro se preferir strict mode)

	case "ssm":
		region := os.Getenv("AWS_REGION")
		val, err := enrichment.ProcessAWSParameterStore(ctx, region, key, true)
		if err != nil {
			return nil, err
		}
		return val, nil

	case "secret":
		region := os.Getenv("AWS_REGION")
		val, err := enrichment.ProcessAWSSecretsManager(ctx, region, key)
		if err != nil {
			return nil, err
		}
		return val, nil
	}

	return nil, nil
}

func setField(field reflect.Value, val interface{}) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(fmt.Sprintf("%v", val))
	}
	return nil
}
