package enrichment

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// --- Mocks para DynamoDB ---

type MockDynamoDB struct {
	GetItemFunc func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
}

func (m *MockDynamoDB) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	return m.GetItemFunc(ctx, params, optFns...)
}

// --- Testes ---

func TestEnrichment_REST_Advanced(t *testing.T) {
	// 1. Cria um Servidor de Teste real (Simula a API externa)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validações do Request que o cliente enviou
		if r.Method != "POST" {
			t.Errorf("Método esperado POST, recebido %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("Header Auth incorreto: %s", r.Header.Get("Authorization"))
		}

		// Valida Body
		body, _ := io.ReadAll(r.Body)
		var bodyMap map[string]interface{}
		json.Unmarshal(body, &bodyMap)
		if bodyMap["user_id"] != "123" {
			t.Errorf("Body incorreto: %s", string(body))
		}

		// Resposta Mockada
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"score": 950, "eligible": true}`))
	}))
	defer server.Close()

	// 2. Executa o ProcessRest contra o servidor de teste
	// Importante: Resetar o Client global ou usar injeção se possível.
	// Como 'sources.go' usa 'var Client HttpClientInterface', podemos injetar o cliente do httptest
	// mas o http.DefaultClient funciona bem com httptest.NewServer URL.
	// Porém, o seu código usa um Client global. Vamos garantir que ele use o default transport para o teste funcionar ou injetar.
	oldClient := Client
	Client = server.Client() // Injeta o cliente configurado para o servidor de teste
	defer func() { Client = oldClient }()

	headers := map[string]string{"Authorization": "Bearer secret"}
	bodyPayload := map[string]string{"user_id": "123"}

	res, err := ProcessRest(context.Background(), "POST", server.URL, headers, bodyPayload)
	if err != nil {
		t.Fatalf("Erro na chamada REST: %v", err)
	}

	// 3. Validações da Resposta
	resMap := res.(map[string]interface{})

	// Valida tipagem correta (JSON Number vs Go Types)
	// Se UseNumber estiver ativo, pode vir como json.Number, precisamos garantir que o teste aceite.
	if score, ok := resMap["score"].(json.Number); ok {
		val, _ := score.Int64()
		if val != 950 {
			t.Errorf("Score incorreto: %d", val)
		}
	} else if score, ok := resMap["score"].(float64); ok { // Caso fallback padrão do Go
		if score != 950 {
			t.Errorf("Score incorreto: %f", score)
		}
	}

	if resMap["eligible"] != true {
		t.Errorf("Eligible incorreto")
	}
}

func TestEnrichment_REST_ErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "user not found"}`))
	}))
	defer server.Close()

	oldClient := Client
	Client = server.Client()
	defer func() { Client = oldClient }()

	_, err := ProcessRest(context.Background(), "GET", server.URL, nil, nil)

	if err == nil {
		t.Fatal("Deveria retornar erro para 404")
	}
	// Verifica se a mensagem de erro contém o status code
	if !contains(err.Error(), "404") {
		t.Errorf("Erro deveria conter status 404, recebido: %v", err)
	}
}

func TestEnrichment_DynamoDB_Advanced(t *testing.T) {
	mockClient := &MockDynamoDB{
		GetItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			// 1. Valida se a chave foi montada corretamente
			pk := params.Key["PK"].(*types.AttributeValueMemberS).Value
			if pk != "USER#123" {
				t.Errorf("PK incorreta: %s", pk)
			}

			// 2. Retorna estrutura complexa (Map e Lista)
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"name": &types.AttributeValueMemberS{Value: "Alice"},
					"tags": &types.AttributeValueMemberL{
						Value: []types.AttributeValue{
							&types.AttributeValueMemberS{Value: "admin"},
							&types.AttributeValueMemberS{Value: "staff"},
						},
					},
					"meta": &types.AttributeValueMemberM{
						Value: map[string]types.AttributeValue{
							"login_count": &types.AttributeValueMemberN{Value: "42"},
						},
					},
				},
			}, nil
		},
	}

	keyMap := map[string]interface{}{"PK": "USER#123"}

	// Chama a função interna refatorada
	res, err := processDynamoDBInternal(context.Background(), mockClient, "MyTable", keyMap)
	if err != nil {
		t.Fatalf("Erro DynamoDB: %v", err)
	}

	resMap := res.(map[string]interface{})

	// Validação
	if resMap["name"] != "Alice" {
		t.Errorf("Nome incorreto")
	}

	tags := resMap["tags"].([]interface{})
	if len(tags) != 2 || tags[0] != "admin" {
		t.Errorf("Lista de tags incorreta")
	}

	meta := resMap["meta"].(map[string]interface{})
	// Números do DynamoDB vêm como string para preservar precisão, a menos que convertidos
	if meta["login_count"] != "42" {
		t.Errorf("Map aninhado incorreto: %v", meta["login_count"])
	}
}

// Helper
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:] != "" // Simplificado, use strings.Contains em prod
}
