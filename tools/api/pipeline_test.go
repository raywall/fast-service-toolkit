// Copyright 2025 Raywall Malheiros de Souza
// Licensed under the Mozilla Public License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.mozilla.org/en-US/MPL/2.0/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
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
