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

// NewHandler cria um único http.Handler (Router) que agrega todas as rotas
// definidas na configuração. Isso satisfaz a chamada no main.go.
func NewHandler(cfg Config) http.Handler {
	router := mux.NewRouter()

	if len(cfg) == 0 {
		// Rota padrão se não houver config
		router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "emulator running (no config)"}`))
		})
		return router
	}

	// Itera sobre todos os servidores definidos no JSON
	for _, srv := range cfg {
		for _, route := range srv.Routes {
			// Usa o método NewHandler definido em route.go para criar a lógica de cada rota
			// srv é passado para manter o contexto se necessário (embora route.go use apenas route config)
			handler := srv.NewHandler(route)

			r := router.HandleFunc(route.Path, handler)
			if route.Method != "" {
				r.Methods(route.Method)
			}

			log.Printf("Rota registrada: [%s] %s", route.Method, route.Path)
		}
	}

	return router
}

// Start (Mantido para compatibilidade, caso usem isoladamente)
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
