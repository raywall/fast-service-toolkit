package enrichment

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// --- Mocks ---

type MockSSM struct {
	GetParameterFunc func(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}

func (m *MockSSM) GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	return m.GetParameterFunc(ctx, params, optFns...)
}

type MockSecrets struct {
	GetSecretValueFunc func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

func (m *MockSecrets) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	return m.GetSecretValueFunc(ctx, params, optFns...)
}

// --- Testes ---

func TestProcessAWSParameterStore(t *testing.T) {
	t.Run("Sucesso", func(t *testing.T) {
		mockVal := "my-config-value"
		mockClient := &MockSSM{
			GetParameterFunc: func(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
				if *params.Name != "/app/config" {
					t.Errorf("Path esperado /app/config, recebido %s", *params.Name)
				}
				if !*params.WithDecryption {
					t.Error("Esperado WithDecryption true")
				}
				return &ssm.GetParameterOutput{
					Parameter: &types.Parameter{Value: &mockVal},
				}, nil
			},
		}

		res, err := getParameterInternal(context.Background(), mockClient, "/app/config", true)
		if err != nil {
			t.Fatalf("Erro inesperado: %v", err)
		}
		if res.(string) != mockVal {
			t.Errorf("Valor incorreto: %v", res)
		}
	})

	t.Run("Erro na AWS", func(t *testing.T) {
		mockClient := &MockSSM{
			GetParameterFunc: func(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
				return nil, errors.New("AWS down")
			},
		}

		_, err := getParameterInternal(context.Background(), mockClient, "/app/config", true)
		if err == nil {
			t.Error("Esperava erro, recebido nil")
		}
	})
}

func TestProcessAWSSecretsManager(t *testing.T) {
	t.Run("Sucesso JSON Parse", func(t *testing.T) {
		secretJSON := `{"api_key": "12345", "host": "localhost"}`
		mockClient := &MockSecrets{
			GetSecretValueFunc: func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				return &secretsmanager.GetSecretValueOutput{
					SecretString: &secretJSON,
				}, nil
			},
		}

		res, err := getSecretInternal(context.Background(), mockClient, "my-secret")
		if err != nil {
			t.Fatalf("Erro inesperado: %v", err)
		}

		resMap := res.(map[string]interface{})
		if resMap["api_key"] != "12345" {
			t.Errorf("JSON parse falhou ou valor incorreto")
		}
	})

	t.Run("Sucesso String Pura", func(t *testing.T) {
		secretStr := "just-a-password"
		mockClient := &MockSecrets{
			GetSecretValueFunc: func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				return &secretsmanager.GetSecretValueOutput{
					SecretString: &secretStr,
				}, nil
			},
		}

		res, err := getSecretInternal(context.Background(), mockClient, "my-secret")
		if err != nil {
			t.Fatalf("Erro inesperado: %v", err)
		}

		if res.(string) != "just-a-password" {
			t.Errorf("Deveria retornar string pura se não for JSON")
		}
	})
}
