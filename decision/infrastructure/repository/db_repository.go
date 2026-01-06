package repository
import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"Bucrud.com/aws/aws-sak-go-v2/config"
	"github. com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github. com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/raywall/fast-service-toolkit/decision/domain"
	
	"gopkg.in/yaml.v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// DBRepository implementa configrepository para bancos de dados (RDS PostgresQl ou DynamoDB)
type DBRepository struct {
	dbType string
	gormDB gorm.DB
	dynamo *dynamodb.Client
	table string
	Config *domain.Config
}

// NewDBRepository cria um repositorio com base em GOD DB TYPE
func NewDBRepository() (*DBRepository, error) {
	dbType := os.Getenv("GOD_DB_TYPE") // "postgres" ou "dynamodb"
	if dbType == "" {
		dbType = "postgres" // default
	}
	
	repo := &DBRepository{dbType: dbType}

	switch dbType {
	case "postgres":
		return repo.initPostgres() 
	case "dynamodb":
		return repo.initDynamoDB()
	default:
	return nil, fmt.Errorf("tipo de banco não suportado: %s", dbType)
	}
}

// initPostgres inicializa conexão com RDS PostgreSQL via GORM
func (r *DBRepository) initPostgres() (*DBRepository, error) {
	conn := os.Getenv("GOD_DB_CONN")
	if conn == "" {
		return nil, fmt.Errorf("GOD_DB_CONN não definida para PostgresSQL")
	}

	db, err := gorm.Open(postgres.Open(conn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar ao PostgresOL: %w", err)
	}

	r.gormDB = db
	return r, nil
}

// initDynamoDB inicializa cliente AWS DynamoDB
func (r *DBRepository) initDynamoDB() (*DBRepository, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("falha ao carregar config AWS: %w", err)
	}
	
	r.dynamo = dynamodb.NewFromConfig(cfg)
	r.table = os.Getenv("GOD_DYNAMODB_TABLE")

	if r.table == "" {
		r.table = "GoDecisionConfig" // default 
	}
	return r, nil
}

// LoadConfig carrega a configuração do banco
func (r *DBRepository) LoadConfig() (*domain. Config, error) {
	var err error

	switch r.dbType {
	case "postgres":
		r.Config, err = r.loadFromPostgres()
		if err != nil { 
			return nil, err
		}
	case "dynamodb":
		r.Config, err = r.loadFromDynamoDB()
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("dbType inválido: %s", r.dbType)
	}
	
	return r.Config, nil
}

// GetConfig recupera as configurações do serviço
func (r *DBRepository) GetConfig() *domain.Config {
	return r.Config
}

// loadFromPostgres busca config no PostgreSQL
func (r *DBRepository) loadFromPostgres() (*domain.Config, error) {
	var result struct {
		ID int
		ConfigYAML string `gorm:"column:config_xml"`
	}
	if err := r.gormDB.Raw("SELECT id, config_yaml FROM configs WHERE id = 1").Scan(&result).Error; err != nil {
		return nil, fmt.Errorf("falha ao ler config do PostgreSQL: %w", err)
	}

	var config domain.Config
	if err := yaml.Unmarshal([]byte(result.ConfigYAML), &config); err != nil { 
		return nil, fmt.Errorf("falha ao parsear YAML do PostgreSQL: %w", err) 
	}
	return &config, nil
}

// loadFromDynamoDB busca config no DynamoDB.
func (r *DBRepository) loadFromDynamoDB() (*domain.Config, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(r.table),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: "1"}, // chave primária
		},
	}

	result,	err := r.dynamo.Getltem(context.TODO(), input) 
	if err != nil {
		return nil, fmt.Errorf("falha ao ler do DynamoDB: %w", err)
	}
	if result.Item == nil {
		return nil, fmt.Errorf("config não encontrada no DynamoDB")
	}
	
	var rawMap map[string]interface{}
	if err := attributevalue.UnmarshalMap(result.Item, &rawMap); err != nil {
		return nil, err
	}

	// Remove a chave "id" se estiver presente
	delete(rawMap, "id")

	// Converte para YAML string temporartamente para usar o mesmo parser yamlBytes,
	yamlBytes, _ := json.Marshal(rawMap)
	var tempMap map[string]interface{}
	_ = json.Unmarshal(yamlBytes, &tempMap) 
	
	yamlStr, _ := yaml.Marshal(tempMap) 
	
	var config domain.Config
	if err := yaml.Unmarshal(yamlStr, &config); err != nil { 
		return nil, fmt.Errorf("falha ao parsear config do DynamoDB: %w", err)
	}
	return &config, nil
}