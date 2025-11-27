package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/raywall/dynamodb-quick-service/api"
)

func main() {
	// Exemplo 1: Pipeline simples sem dependências
	exemploSimples()

	// Exemplo 2: Pipeline com dependências
	exemploComDependencias()

	// Exemplo 3: Pipeline com autenticação (Token Service)
	exemploComAutenticacao()

	// Exemplo 4: Pipeline com API obrigatória
	exemploComAPIObrigatoria()
}

// exemploSimples demonstra execução paralela de múltiplas APIs independentes
func exemploSimples() {
	fmt.Println("\n=== Exemplo 1: Pipeline Simples ===")

	// Configuração das APIs
	apis := []api.APIConfig{
		{
			Name:     "UserAPI",
			Required: false,
			Parameters: api.APIParameters{
				HttpMethod: http.MethodGet,
				Host:       "https://jsonplaceholder.typicode.com/users/1",
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
			},
		},
		{
			Name:     "PostsAPI",
			Required: false,
			Parameters: api.APIParameters{
				HttpMethod: http.MethodGet,
				Host:       "https://jsonplaceholder.typicode.com/posts/1",
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
			},
		},
		{
			Name:     "CommentsAPI",
			Required: false,
			Parameters: api.APIParameters{
				HttpMethod: http.MethodGet,
				Host:       "https://jsonplaceholder.typicode.com/comments/1",
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
			},
		},
	}

	// Cria o pipeline
	pipeline := api.NewAPIPipeline(apis)

	// Executa com timeout de 10 segundos
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results, err := pipeline.Execute(ctx, nil)
	if err != nil {
		log.Printf("Erro na execução: %v", err)
		return
	}

	// Exibe os resultados
	fmt.Println("\nResultados:")
	for name, data := range results {
		jsonData, _ := json.MarshalIndent(data, "", "  ")
		fmt.Printf("\n%s:\n%s\n", name, string(jsonData))
	}
}

// exemploComDependencias demonstra execução com dependências entre APIs
func exemploComDependencias() {
	fmt.Println("\n=== Exemplo 2: Pipeline com Dependências ===")

	apis := []api.APIConfig{
		// API 1: Busca usuário (sem dependências)
		{
			Name:     "GetUser",
			Required: true,
			Parameters: api.APIParameters{
				HttpMethod: http.MethodGet,
				Host:       "https://jsonplaceholder.typicode.com/users/1",
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
			},
		},
		// API 2: Busca posts do usuário (depende de GetUser)
		{
			Name:         "GetUserPosts",
			Required:     false,
			Dependencies: []string{"GetUser"},
			Parameters: api.APIParameters{
				HttpMethod: http.MethodGet,
				Host:       "https://jsonplaceholder.typicode.com/users/1/posts",
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
			},
		},
		// API 3: Busca álbuns do usuário (depende de GetUser)
		{
			Name:         "GetUserAlbums",
			Required:     false,
			Dependencies: []string{"GetUser"},
			Parameters: api.APIParameters{
				HttpMethod: http.MethodGet,
				Host:       "https://jsonplaceholder.typicode.com/users/1/albums",
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
			},
		},
	}

	pipeline := api.NewAPIPipeline(apis)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results, err := pipeline.Execute(ctx, nil)
	if err != nil {
		log.Printf("Erro na execução: %v", err)
		return
	}

	fmt.Printf("\nTotal de APIs executadas: %d\n", len(results))
	for name := range results {
		fmt.Printf("- %s: ✓\n", name)
	}
}

// exemploComAutenticacao demonstra uso com Token Service
func exemploComAutenticacao() {
	fmt.Println("\n=== Exemplo 3: Pipeline com Autenticação ===")

	// Configura o serviço de token
	tokenService := api.NewTokenService()
	tokenService.Configurations["my-service"] = api.TokenConfig{
		GrantType:    "client_credentials",
		ClientID:     "my-client-id",
		ClientSecret: "my-client-secret",
		Host:         "https://auth.example.com/oauth/token",
		Httpmethod:   http.MethodPost,
	}

	// Obtém o token (em produção, faça tratamento de erro adequado)
	token, err := tokenService.GetToken("my-service")
	if err != nil {
		log.Printf("Erro ao obter token: %v", err)
		// Para o exemplo, vamos usar um token mock
		mockToken := "mock-token-123"
		token = &mockToken
	}

	// Configura APIs que usam o token
	apis := []api.APIConfig{
		{
			Name:     "ProtectedAPI1",
			Required: true,
			Parameters: api.APIParameters{
				AccessToken: token,
				HttpMethod:  http.MethodGet,
				Host:        "https://api.example.com/protected/resource1",
				Headers: map[string]string{
					"Content-Type":  "application/json",
					"Authorization": fmt.Sprintf("Bearer %s", *token),
				},
			},
		},
		{
			Name:     "ProtectedAPI2",
			Required: false,
			Parameters: api.APIParameters{
				AccessToken: token,
				HttpMethod:  http.MethodGet,
				Host:        "https://api.example.com/protected/resource2",
				Headers: map[string]string{
					"Content-Type":  "application/json",
					"Authorization": fmt.Sprintf("Bearer %s", *token),
				},
			},
		},
	}

	pipeline := api.NewAPIPipeline(apis)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results, err := pipeline.Execute(ctx, nil)
	if err != nil {
		log.Printf("Erro na execução: %v", err)
		return
	}

	fmt.Printf("\nAPIs autenticadas executadas: %d\n", len(results))
}

// exemploComAPIObrigatoria demonstra comportamento com APIs obrigatórias
func exemploComAPIObrigatoria() {
	fmt.Println("\n=== Exemplo 4: Pipeline com API Obrigatória ===")

	apis := []api.APIConfig{
		// API obrigatória - se falhar, todo o pipeline falha
		{
			Name:     "CriticalAPI",
			Required: true,
			Parameters: api.APIParameters{
				HttpMethod: http.MethodGet,
				Host:       "https://jsonplaceholder.typicode.com/users/1",
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
			},
		},
		// API opcional - pode falhar sem afetar o pipeline
		{
			Name:     "OptionalAPI",
			Required: false,
			Parameters: api.APIParameters{
				HttpMethod: http.MethodGet,
				Host:       "https://jsonplaceholder.typicode.com/invalid-endpoint",
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
			},
		},
	}

	pipeline := api.NewAPIPipeline(apis)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results, err := pipeline.Execute(ctx, nil)
	if err != nil {
		log.Printf("Pipeline falhou: %v", err)
		return
	}

	fmt.Println("\nAPIs executadas com sucesso:")
	for name := range results {
		fmt.Printf("- %s\n", name)
	}
}

// exemploComplexo demonstra um caso de uso real com múltiplas dependências
func exemploComplexo() {
	fmt.Println("\n=== Exemplo 5: Caso de Uso Complexo ===")

	// Cenário: Sistema de e-commerce
	// 1. Autentica
	// 2. Busca dados do usuário
	// 3. Em paralelo: busca carrinho, histórico de pedidos, wishlist
	// 4. Calcula recomendações baseado no histórico

	// tokenService := api.NewTokenService()
	// ... configuração do token service

	apis := []api.APIConfig{
		// Nível 0: Sem dependências
		{
			Name:     "GetUser",
			Required: true,
			Parameters: api.APIParameters{
				HttpMethod: http.MethodGet,
				Host:       "https://api.ecommerce.com/users/me",
				Headers:    map[string]string{"Content-Type": "application/json"},
			},
		},
		// Nível 1: Dependem de GetUser
		{
			Name:         "GetCart",
			Required:     false,
			Dependencies: []string{"GetUser"},
			Parameters: api.APIParameters{
				HttpMethod: http.MethodGet,
				Host:       "https://api.ecommerce.com/cart",
				Headers:    map[string]string{"Content-Type": "application/json"},
			},
		},
		{
			Name:         "GetOrderHistory",
			Required:     false,
			Dependencies: []string{"GetUser"},
			Parameters: api.APIParameters{
				HttpMethod: http.MethodGet,
				Host:       "https://api.ecommerce.com/orders/history",
				Headers:    map[string]string{"Content-Type": "application/json"},
			},
		},
		{
			Name:         "GetWishlist",
			Required:     false,
			Dependencies: []string{"GetUser"},
			Parameters: api.APIParameters{
				HttpMethod: http.MethodGet,
				Host:       "https://api.ecommerce.com/wishlist",
				Headers:    map[string]string{"Content-Type": "application/json"},
			},
		},
		// Nível 2: Depende do histórico de pedidos
		{
			Name:         "GetRecommendations",
			Required:     false,
			Dependencies: []string{"GetOrderHistory"},
			Parameters: api.APIParameters{
				HttpMethod: http.MethodGet,
				Host:       "https://api.ecommerce.com/recommendations",
				Headers:    map[string]string{"Content-Type": "application/json"},
			},
		},
	}

	pipeline := api.NewAPIPipeline(apis)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	start := time.Now()
	results, err := pipeline.Execute(ctx, nil)
	duration := time.Since(start)

	if err != nil {
		log.Printf("Pipeline falhou: %v", err)
		return
	}

	fmt.Printf("\nPipeline executado em %v\n", duration)
	fmt.Printf("APIs concluídas: %d\n", len(results))
	fmt.Println("\nOrdem de execução esperada:")
	fmt.Println("1. GetUser (paralelo, sem dependências)")
	fmt.Println("2. GetCart, GetOrderHistory, GetWishlist (paralelo, dependem de GetUser)")
	fmt.Println("3. GetRecommendations (depende de GetOrderHistory)")
}