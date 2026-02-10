package engine

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// --- Mocks ---

type MockS3Loader struct {
	GetObjectFunc func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

func (m *MockS3Loader) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return m.GetObjectFunc(ctx, params, optFns...)
}

type MockDynamoLoader struct {
	GetItemFunc func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
}

func (m *MockDynamoLoader) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	return m.GetItemFunc(ctx, params, optFns...)
}

// --- Testes ---

func TestUniversalLoader_Load_Local(t *testing.T) {
	// YAML Válido e Completo para passar na validação estrita
	yamlContent := `
version: "1.0"
service:
  name: "test-local"
  runtime: "local"
  port: 8080
  route: "/test"
  timeout: "1s"
  # CORREÇÃO: Campos obrigatórios de Timeout
  on_timeout:
    code: 504
    msg: "Gateway Timeout"
  # CORREÇÃO: Campos obrigatórios de Logging (mesmo disabled, o validador checa formato)
  logging:
    enabled: false
    level: "info"   # required oneof=debug info warn error
    format: "json"  # required oneof=json console
  metrics: 
    datadog: 
      enabled: false

steps:
  input: {}
  processing: {}
  output: 
    status_code: 200 
    body: {}
`
	tmpFile, _ := os.CreateTemp("", "config_*.yaml")
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(yamlContent); err != nil {
		t.Fatalf("Erro ao escrever arquivo: %v", err)
	}
	tmpFile.Close()

	loader := NewUniversalLoader()
	cfg, err := loader.Load(context.Background(), tmpFile.Name()) // Teste implícito de file://

	if err != nil {
		t.Fatalf("Erro load local: %v", err)
	}
	if cfg.Service.Name != "test-local" {
		t.Errorf("Nome incorreto: %s", cfg.Service.Name)
	}
}

func TestUniversalLoader_S3_Internal(t *testing.T) {
	mockYaml := `version: "1.0"`
	mockClient := &MockS3Loader{
		GetObjectFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			if *params.Bucket != "my-bucket" || *params.Key != "configs/svc.yaml" {
				t.Errorf("Params S3 incorretos: %v", params)
			}
			return &s3.GetObjectOutput{
				Body: io.NopCloser(strings.NewReader(mockYaml)),
			}, nil
		},
	}

	loader := NewUniversalLoader()
	// Chama o método interno injetando o mock
	data, err := loader.loadFromS3Internal(context.Background(), mockClient, "s3://my-bucket/configs/svc.yaml")

	if err != nil {
		t.Fatalf("Erro s3 internal: %v", err)
	}
	if string(data) != mockYaml {
		t.Errorf("Conteúdo incorreto")
	}
}

func TestUniversalLoader_Dynamo_Internal(t *testing.T) {
	mockClient := &MockDynamoLoader{
		GetItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			if *params.TableName != "ConfigTable" {
				t.Errorf("Tabela incorreta: %s", *params.TableName)
			}
			// Verifica query params customizados
			key := params.Key["ServiceName"].(*types.AttributeValueMemberS).Value
			if key != "my-svc" {
				t.Errorf("PK incorreta: %s", key)
			}

			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"yaml_body": &types.AttributeValueMemberS{Value: `version: "1.0"`},
				},
			}, nil
		},
	}

	loader := NewUniversalLoader()
	// URI complexa: Tabela=ConfigTable, PK_Value=my-svc, PK_Name=ServiceName, Col=yaml_body
	uri := "dynamodb://ConfigTable/my-svc?pk=ServiceName&col=yaml_body"

	data, err := loader.loadFromDynamoDBInternal(context.Background(), mockClient, uri)

	if err != nil {
		t.Fatalf("Erro dynamo internal: %v", err)
	}
	if string(data) != `version: "1.0"` {
		t.Errorf("Conteúdo incorreto")
	}
}
