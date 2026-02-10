package enrichment

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
)

// MockClient para evitar chamadas reais
type MockClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

func TestProcessRest(t *testing.T) {
	// Setup Mock
	mockResponse := `{"id": 1, "status": "ok"}`
	Client = &MockClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(mockResponse)),
			}, nil
		},
	}

	res, err := ProcessRest(context.Background(), "GET", "http://fake.com", nil, nil)
	if err != nil {
		t.Fatalf("Erro inesperado: %v", err)
	}

	data := res.(map[string]interface{})
	if data["status"] != "ok" {
		t.Errorf("Esperado status ok, recebido %v", data["status"])
	}
}

func TestProcessGraphQL(t *testing.T) {
	// Setup Mock
	mockResponse := `{"data": {"user": {"name": "Raywall"}}}`
	Client = &MockClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(mockResponse)),
			}, nil
		},
	}

	res, err := ProcessGraphQL(context.Background(), "http://gql.com", "query...", nil, nil)
	if err != nil {
		t.Fatalf("Erro GraphQL: %v", err)
	}

	data := res.(map[string]interface{})
	user := data["user"].(map[string]interface{})
	if user["name"] != "Raywall" {
		t.Errorf("Esperado Raywall, recebido %v", user["name"])
	}
}
