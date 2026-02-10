package enrichment

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"gopkg.in/yaml.v3"
)

// S3Client interface para Mock
type S3Client interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

func ProcessS3(ctx context.Context, region, bucket, key, format string) (interface{}, error) {
	cfg, err := GetAWSConfig(ctx, region)
	if err != nil {
		return nil, err
	}
	return processS3Internal(ctx, s3.NewFromConfig(cfg), bucket, key, format)
}

// processS3Internal contém a lógica de download e parse
func processS3Internal(ctx context.Context, client S3Client, bucket, key, format string) (interface{}, error) {
	out, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, fmt.Errorf("erro ao baixar do S3: %w", err)
	}
	defer out.Body.Close()

	bodyBytes, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(format) {
	case "json":
		var result interface{}
		if err := json.Unmarshal(bodyBytes, &result); err != nil {
			return nil, fmt.Errorf("erro parse JSON s3: %w", err)
		}
		return result, nil
	case "yaml", "yml":
		var result interface{}
		if err := yaml.Unmarshal(bodyBytes, &result); err != nil {
			return nil, fmt.Errorf("erro parse YAML s3: %w", err)
		}
		return result, nil
	case "csv":
		reader := csv.NewReader(strings.NewReader(string(bodyBytes)))
		records, err := reader.ReadAll()
		if err != nil {
			return nil, fmt.Errorf("erro parse CSV s3: %w", err)
		}
		return parseCSVToMap(records), nil
	default:
		return string(bodyBytes), nil
	}
}

func parseCSVToMap(records [][]string) []map[string]interface{} {
	if len(records) < 1 {
		return nil
	}
	headers := records[0]
	var result []map[string]interface{}

	for _, row := range records[1:] {
		item := make(map[string]interface{})
		for i, val := range row {
			if i < len(headers) {
				item[headers[i]] = val
			}
		}
		result = append(result, item)
	}
	return result
}
