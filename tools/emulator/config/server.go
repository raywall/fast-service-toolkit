package config

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// ServerConfig para cada servidor/porta
type ServerConfig struct {
	Port   int           `json:"port"`
	Routes []RouteConfig `json:"routes"`
}

func (s *ServerConfig) Start() {
	router := mux.NewRouter()
	for _, route := range s.Routes {
		handler := s.NewHandler(route)
		router.HandleFunc(route.Path, handler).Methods(route.Method)
	}

	addr := fmt.Sprintf(":%d", s.Port)
	log.Printf("Iniciando servidor na porta %d", s.Port)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Printf("Erro no servidor porta %d: %v", s.Port, err)
	}
}
