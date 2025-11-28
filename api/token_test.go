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
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetToken_Success(t *testing.T) {
	// Configuração do servidor de teste
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("Expected Content-Type header to be application/x-www-form-urlencoded, got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token": "mock_token"}`))
	}))
	defer mockServer.Close()

	// Configuração do TokenService
	tokenService := NewTokenService()
	tokenService.Configurations["test"] = TokenConfig{
		GrantType:    "client_credentials",
		ClientID:     "test_client",
		ClientSecret: "test_secret",
		Host:         mockServer.URL,
		HttpMethod:   http.MethodPost,
	}

	// Execução do teste
	token, err := tokenService.GetToken("test")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if *token != "mock_token" {
		t.Errorf("Expected token to be 'mock_token', got %s", *token)
	}
}

func TestGetToken_NotFoud(t *testing.T) {
	tokenService := NewTokenService()

	_, err := tokenService.GetToken("invalid")
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	if err.Error() != ErrNotFound.Error() {
		t.Errorf("Expected error '%v', got '%v'", ErrNotFound, err)
	}
}

func TestGetToken_InvalidResponse(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`invalid_json`))
	}))
	defer mockServer.Close()

	tokenService := NewTokenService()
	tokenService.Configurations["test"] = TokenConfig{
		GrantType:    "client_credentials",
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		Host:         mockServer.URL,
		HttpMethod:   http.MethodPost,
	}

	_, err := tokenService.GetToken("test")
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
}
