package enrichment

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// Interfaces para abstrair o SDK da AWS (Permite Mocking)
type SSMClient interface {
	GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}

type SecretsClient interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

// ProcessAWSParameterStore: Wrapper público que inicializa o client real.
func ProcessAWSParameterStore(ctx context.Context, region, path string, decrypt bool) (interface{}, error) {
	cfg, err := GetAWSConfig(ctx, region)
	if err != nil {
		return nil, err
	}
	// Injeta o cliente real na lógica interna
	return getParameterInternal(ctx, ssm.NewFromConfig(cfg), path, decrypt)
}

// getParameterInternal: Lógica pura testável via Mock.
func getParameterInternal(ctx context.Context, client SSMClient, path string, decrypt bool) (interface{}, error) {
	out, err := client.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           &path,
		WithDecryption: &decrypt,
	})
	if err != nil {
		return nil, fmt.Errorf("erro no SSM GetParameter: %w", err)
	}
	return *out.Parameter.Value, nil
}

// ProcessAWSSecretsManager: Wrapper público que inicializa o client real.
func ProcessAWSSecretsManager(ctx context.Context, region, secretID string) (interface{}, error) {
	cfg, err := GetAWSConfig(ctx, region)
	if err != nil {
		return nil, err
	}
	return getSecretInternal(ctx, secretsmanager.NewFromConfig(cfg), secretID)
}

// getSecretInternal: Lógica pura testável via Mock.
func getSecretInternal(ctx context.Context, client SecretsClient, secretID string) (interface{}, error) {
	out, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &secretID,
	})
	if err != nil {
		return nil, fmt.Errorf("erro no SecretsManager: %w", err)
	}

	val := *out.SecretString

	// Tenta decodificar JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(val), &data); err == nil {
		return data, nil
	}
	return val, nil
}
