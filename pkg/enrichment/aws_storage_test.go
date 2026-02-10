package enrichment

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type MockS3 struct {
	GetObjectFunc func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

func (m *MockS3) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return m.GetObjectFunc(ctx, params, optFns...)
}

func TestProcessS3(t *testing.T) {
	t.Run("Sucesso JSON", func(t *testing.T) {
		jsonContent := `{"name": "test"}`
		mockClient := &MockS3{
			GetObjectFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				return &s3.GetObjectOutput{
					Body: io.NopCloser(strings.NewReader(jsonContent)),
				}, nil
			},
		}

		res, err := processS3Internal(context.Background(), mockClient, "bucket", "file.json", "json")
		if err != nil {
			t.Fatalf("Erro inesperado: %v", err)
		}

		resMap := res.(map[string]interface{})
		if resMap["name"] != "test" {
			t.Error("Parse JSON incorreto")
		}
	})

	t.Run("Sucesso CSV", func(t *testing.T) {
		csvContent := "id,name\n1,raywall"
		mockClient := &MockS3{
			GetObjectFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				return &s3.GetObjectOutput{
					Body: io.NopCloser(strings.NewReader(csvContent)),
				}, nil
			},
		}

		res, err := processS3Internal(context.Background(), mockClient, "bucket", "file.csv", "csv")
		if err != nil {
			t.Fatalf("Erro inesperado: %v", err)
		}

		resSlice := res.([]map[string]interface{})
		if len(resSlice) != 1 {
			t.Fatal("Deveria ter 1 registro")
		}
		if resSlice[0]["name"] != "raywall" {
			t.Error("Parse CSV incorreto")
		}
	})
}
