package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAPIPipeline_Execute_Success(t *testing.T) {
	// Configuração do servidor de teste mock
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result": "success"}`))
	}))
	defer mockServer.Close()

	// Configuração das APIs
	apiConfig := APIConfig{
		Name:     "TestAPI",
		Required: true,
		Parameters: APIParameters{
			HttpMethod: http.MethodGet,
			Host:       mockServer.URL,
			Headers:    map[string]string{"Content-Type": "application/json"},
		},
	}

	pipeline := NewAPIPipeline([]APIConfig{apiConfig})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results, err := pipeline.Execute(ctx, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result, ok := results["TestAPI"]; !ok || result.(map[string]interface{})["result"] != "success" {
		t.Errorf("Expected result to be 'Success', got %v", result)
	}
}

func TestAPIPipeline_Execute_Failure(t *testing.T) {
	// Configuração do servidor de teste mock com erro
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	apiConfig := APIConfig{
		Name:     "TestAPI",
		Required: true,
		Parameters: APIParameters{
			HttpMethod: http.MethodGet,
			Host:       mockServer.URL,
			Headers:    map[string]string{"Content-Type": "application/json"},
		},
	}
	pipeline := NewAPIPipeline([]APIConfig{apiConfig})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results, err := pipeline.Execute(ctx, nil)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	if results != nil {
		t.Errorf("Expected results to be nil, got %v", results)
	}
}

func TestAPIPipeline_Execute_WithDependencies(t *testing.T) {
	// Configuração dos servidores de teste mock
	mockServer1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result": "dependency success"}`))
	}))
	defer mockServer1.Close()

	mockServer2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result": "main success"}`))
	}))
	defer mockServer2.Close()

	apiDependency := APIConfig{
		Name:     "DependencyAPI",
		Required: false,
		Parameters: APIParameters{
			HttpMethod: http.MethodGet,
			Host:       mockServer1.URL,
			Headers:    map[string]string{"Content-Type": "application/json"},
		},
	}

	apiMain := APIConfig{
		Name:         "MainAPI",
		Required:     true,
		Dependencies: []string{"DependencyAPI"},
		Parameters: APIParameters{
			HttpMethod: http.MethodGet,
			Host:       mockServer2.URL,
			Headers:    map[string]string{"Content-Type": "application/json"},
		},
	}

	pipeline := NewAPIPipeline([]APIConfig{apiDependency, apiMain})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results, err := pipeline.Execute(ctx, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result, ok := results["MainAPI"]; !ok || result.(map[string]interface{})["result"] != "main success" {
		t.Errorf("Expected result to be 'main success', got %v", result)
	}
}
