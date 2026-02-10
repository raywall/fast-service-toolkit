package config

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/raywall/fast-service-toolkit/tools/emulator/types"
)

// RouteConfig para cada rota
type RouteConfig struct {
	Path              string               `json:"path"`
	Method            string               `json:"method"`
	Response          *types.Response      `json:"response,omitempty"` // Para respostas estáticas
	Data              []interface{}        `json:"data,omitempty"`     // Para dados dinâmicos
	QueryParams       []types.ParamMapping `json:"query_params,omitempty"`
	PathParams        []types.ParamMapping `json:"path_params,omitempty"`
	ResponseOnMatch   *types.Response      `json:"response_on_match,omitempty"`
	ResponseOnNoMatch *types.Response      `json:"response_on_no_match,omitempty"`
}

func (s *ServerConfig) NewHandler(route RouteConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Se for resposta estática (sem data/params)
		if route.Response != nil && len(route.Data) == 0 && len(route.QueryParams) == 0 && len(route.PathParams) == 0 {
			sendResponse(w, route.Response.Status, route.Response.Body)
			return
		}

		// Caso contrário, dinâmico: filtrar data baseado em params
		params := make(map[string]string)

		// Extrair path params (via mux)
		vars := mux.Vars(r)
		for _, p := range route.PathParams {
			value, ok := vars[p.Name]
			if ok {
				params[p.MapsTo] = value
			}
		}

		// Extrair query params
		query := r.URL.Query()
		for _, p := range route.QueryParams {
			value := query.Get(p.Name)
			if value != "" {
				params[p.MapsTo] = value
			}
		}

		// Filtrar data (assumindo data é []map[string]interface{})
		var matches []interface{}
		for _, item := range route.Data {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue // Skip se não for map
			}
			match := true
			for field, value := range params {
				itemValue, exists := itemMap[field]
				if !exists || !valuesMatch(itemValue, value) {
					match = false
					break
				}
			}
			if match {
				matches = append(matches, item)
			}
		}

		if len(matches) == 0 {
			resp := route.ResponseOnNoMatch
			if resp == nil {
				resp = &types.Response{Status: 404, Body: map[string]string{"error": "Not found"}}
			}
			sendResponse(w, resp.Status, resp.Body)
			return
		}

		resp := route.ResponseOnMatch
		if resp == nil {
			resp = &types.Response{Status: 200}
		}

		var body interface{}
		if len(matches) == 1 {
			body = matches[0]
		} else {
			body = matches
		}

		sendResponse(w, resp.Status, body)
	}
}

func sendResponse(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body != nil {
		if err := json.NewEncoder(w).Encode(body); err != nil {
			log.Printf("Erro ao encode response: %v", err)
		}
	}
}

func valuesMatch(a interface{}, b string) bool {
	switch v := a.(type) {
	case string:
		return v == b
	case float64:
		f, err := strconv.ParseFloat(b, 64)
		return err == nil && v == f
	case int:
		i, err := strconv.Atoi(b)
		return err == nil && v == i
	case bool:
		return strings.ToLower(b) == fmt.Sprintf("%v", v)
	default:
		return false
	}
}
