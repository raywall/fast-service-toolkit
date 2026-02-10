package config

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/raywall/fast-service-toolkit/tools/emulator/types"
)

// helper para executar request através do roteador (necessário para mux.Vars funcionar)
func executeRequest(handler http.HandlerFunc, method, path string) *httptest.ResponseRecorder {
	router := mux.NewRouter()
	// Registra o handler com suporte a path vars se necessário
	router.HandleFunc("/users/{id}", handler).Methods(method)
	router.HandleFunc("/users", handler).Methods(method) // Fallback para lista

	req, _ := http.NewRequest(method, path, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func TestNewHandler_StaticResponse(t *testing.T) {
	cfg := &ServerConfig{}
	route := RouteConfig{
		Path:     "/users",
		Method:   "GET",
		Response: &types.Response{Status: 200, Body: map[string]string{"msg": "static"}},
	}

	handler := cfg.NewHandler(route)
	rr := executeRequest(handler, "GET", "/users")

	if rr.Code != 200 {
		t.Errorf("Status esperado 200, recebido %d", rr.Code)
	}
	expected := `{"msg":"static"}`
	if rr.Body.String() != expected+"\n" { // json.Encoder adiciona newline
		t.Errorf("Body incorreto: %s", rr.Body.String())
	}
}

func TestNewHandler_Dynamic_PathParams(t *testing.T) {
	cfg := &ServerConfig{}

	// Dados Mockados
	data := []interface{}{
		map[string]interface{}{"id": 1, "name": "Alice"},
		map[string]interface{}{"id": 2, "name": "Bob"},
	}

	route := RouteConfig{
		Path:   "/users/{id}",
		Method: "GET",
		Data:   data,
		PathParams: []types.ParamMapping{
			{Name: "id", MapsTo: "id"},
		},
		ResponseOnMatch:   &types.Response{Status: 200},
		ResponseOnNoMatch: &types.Response{Status: 404, Body: "not found"},
	}

	handler := cfg.NewHandler(route)

	t.Run("Match Found (Alice)", func(t *testing.T) {
		rr := executeRequest(handler, "GET", "/users/1")
		if rr.Code != 200 {
			t.Errorf("Esperado 200, recebido %d", rr.Code)
		}
		var res map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &res)
		if res["name"] != "Alice" {
			t.Errorf("Esperado Alice, recebido %v", res["name"])
		}
	})

	t.Run("No Match", func(t *testing.T) {
		rr := executeRequest(handler, "GET", "/users/999")
		if rr.Code != 404 {
			t.Errorf("Esperado 404, recebido %d", rr.Code)
		}
	})
}

func TestNewHandler_Dynamic_QueryParams(t *testing.T) {
	cfg := &ServerConfig{}
	data := []interface{}{
		map[string]interface{}{"type": "admin", "name": "Alice"},
		map[string]interface{}{"type": "user", "name": "Bob"},
	}

	route := RouteConfig{
		Path:   "/users",
		Method: "GET",
		Data:   data,
		QueryParams: []types.ParamMapping{
			{Name: "role", MapsTo: "type"},
		},
	}

	handler := cfg.NewHandler(route)

	// GET /users?role=admin
	req, _ := http.NewRequest("GET", "/users?role=admin", nil)
	rr := httptest.NewRecorder()

	// Não precisamos de mux router aqui pois query params são standard lib
	handler.ServeHTTP(rr, req)

	if rr.Code != 200 { // Default status code
		t.Errorf("Esperado 200, recebido %d", rr.Code)
	}

	// var res []interface{} // Pode retornar lista ou objeto único, lógica diz: 1 match = objeto
	// No código original: if len(matches) == 1 { body = matches[0] }
	var resObj map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resObj)

	if resObj["name"] != "Alice" {
		t.Errorf("Esperado Alice, recebido %v", resObj)
	}
}
