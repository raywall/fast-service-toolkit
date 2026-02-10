package engine

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	localConfig "github.com/raywall/fast-service-toolkit/pkg/config"
	"github.com/raywall/fast-service-toolkit/pkg/config/injector"
	"gopkg.in/yaml.v2"
)

// --- MÉTODOS PÚBLICOS DE PACOTE (Para corrigir o erro undefined: config.Load) ---

// Load é a função simplificada que o ServiceEngine chama.
// Ela abstrai a criação do UniversalLoader.
func Load(source string) (*localConfig.ServiceConfig, error) {
	loader := NewUniversalLoader()
	// Usamos context.Background() pois geralmente o Load ocorre na inicialização
	// ou em uma goroutine de background (Hot Reload)
	return loader.Load(context.Background(), source)
}

// --- Interfaces para Mocking ---

type S3Downloader interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

type DynamoGetter interface {
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
}

// UniversalLoader suporta múltiplas fontes de configuração (Local, S3, DynamoDB).
type UniversalLoader struct {
	validator *localConfig.ConfigValidator
}

// --- UniversalLoader ---

// NewUniversalLoader cria uma nova instância.
func NewUniversalLoader() *UniversalLoader {
	return &UniversalLoader{
		validator: localConfig.NewValidator(),
	}
}

// Load detecta o esquema da fonte e carrega a configuração.
func (ul *UniversalLoader) Load(ctx context.Context, source string) (*localConfig.ServiceConfig, error) {
	var rawData []byte
	var err error

	if strings.HasPrefix(source, "s3://") {
		// Inicializa cliente real S3
		cfg, _ := config.LoadDefaultConfig(ctx)
		client := s3.NewFromConfig(cfg)
		rawData, err = ul.loadFromS3Internal(ctx, client, source)

	} else if strings.HasPrefix(source, "dynamodb://") {
		// Inicializa cliente real DynamoDB
		cfg, _ := config.LoadDefaultConfig(ctx)
		client := dynamodb.NewFromConfig(cfg)
		rawData, err = ul.loadFromDynamoDBInternal(ctx, client, source)

	} else {
		// Default: Arquivo Local
		rawData, err = ul.loadFromFile(source)
	}

	if err != nil {
		return nil, fmt.Errorf("falha leitura config (%s): %w", source, err)
	}

	return ul.parseAndValidate(ctx, rawData)
}

// --- Estratégias de carregamento (métodos internos testáveis) ---

func (ul *UniversalLoader) loadFromFile(path string) ([]byte, error) {
	// Suporta tanto "file://config.yaml" quanto apenas "config.yaml"
	cleanPath := strings.TrimPrefix(path, "file://")
	return os.ReadFile(cleanPath)
}

func (ul *UniversalLoader) loadFromS3Internal(ctx context.Context, client S3Downloader, uri string) ([]byte, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("URL S3 inválida: %w", err)
	}
	bucket := u.Host
	key := strings.TrimPrefix(u.Path, "/")

	out, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()

	return io.ReadAll(out.Body)
}

func (ul *UniversalLoader) loadFromDynamoDBInternal(ctx context.Context, client DynamoGetter, uri string) ([]byte, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("URL DynamoDB inválida: %w", err)
	}

	tableName := u.Host
	pkValue := strings.TrimPrefix(u.Path, "/")

	// Query Params opcionais: dynamodb://tabela/chave?col=dado&pk=UserId
	colName := u.Query().Get("col")
	if colName == "" {
		colName = "config" // Coluna padrão onde o YAML está salvo
	}

	pkName := u.Query().Get("pk")
	if pkName == "" {
		pkName = "id" // Nome padrão da Partition Key
	}

	keyMap := map[string]types.AttributeValue{
		pkName: &types.AttributeValueMemberS{Value: pkValue},
	}

	out, err := client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &tableName,
		Key:       keyMap,
	})
	if err != nil {
		return nil, err
	}

	if out.Item == nil {
		return nil, fmt.Errorf("item não encontrado no DynamoDB")
	}

	var itemMap map[string]interface{}
	if err := attributevalue.UnmarshalMap(out.Item, &itemMap); err != nil {
		return nil, err
	}

	content, ok := itemMap[colName].(string)
	if !ok {
		return nil, fmt.Errorf("coluna '%s' inválida ou vazia no DynamoDB", colName)
	}

	return []byte(content), nil
}

// parseAndValidate agora aceita context para passar ao Injector
func (ul *UniversalLoader) parseAndValidate(ctx context.Context, data []byte) (*localConfig.ServiceConfig, error) {
	var cfg localConfig.ServiceConfig

	// 1. Unmarshal (YAML -> Struct)
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("YAML malformado: %w", err)
	}

	// 2. Injection (Env/Secrets/SSM)
	// Usa o Injector para resolver valores como env://DB_PASS
	inj := injector.New()
	if err := inj.Inject(ctx, &cfg); err != nil {
		return nil, fmt.Errorf("falha na injeção de variáveis: %w", err)
	}

	// 3. Validation
	if ul.validator != nil {
		if err := ul.validator.Validate(&cfg); err != nil {
			return nil, fmt.Errorf("validação da configuração falhou: %w", err)
		}
	}

	return &cfg, nil
}
