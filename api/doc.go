// Package api fornece um framework para orquestrar chamadas de API externas
// em um pipeline concorrente, respeitando dependências, e utilitários para
// obtenção de tokens de segurança (STS).
//
// Visão Geral:
// O pacote `api` foi projetado para gerenciar a complexidade de sistemas que
// precisam chamar múltiplas APIs de terceiros ou microserviços internos.
// Ele utiliza a estrutura APIPipeline para executar essas chamadas em paralelo
// e garante que nenhuma API seja executada antes que todas as suas dependências
// tenham retornado um resultado. Ele também inclui um serviço de Token (STS)
// para gerenciar a autenticação nessas APIs.
//
// Funcionalidades Principais:
//   - APIPipeline: Execução concorrente de múltiplas chamadas HTTP com gerenciamento
//     de dependências, usando a lógica de um grafo dirigido acíclico (DAG).
//   - Circuit Breaker: APIs marcadas como Required cancelam todo o pipeline
//     (erros 422) se falharem.
//   - TokenService: Simplifica a obtenção e gerenciamento de tokens de acesso (STS).
//
// Exemplos de Uso:
//
// Exemplo Básico de Pipeline:
// Demonstra como configurar e executar um pipeline de APIs.
//
//	configA := api.APIConfig{Name: "User", Host: "https://user-api/v1/user", HttpMethod: "GET"}
//	configB := api.APIConfig{
//		Name: "Profile",
//		Dependencies: []string{"User"}, // Depende de 'User'
//		Host: "https://profile-api/v1/profile",
//		HttpMethod: "POST",
//	}
//
//	pipeline := api.NewAPIPipeline([]api.APIConfig{configA, configB})
//
//	results, err := pipeline.Execute(context.Background(), nil)
//	if err != nil {
//		// Tratar erro do pipeline
//	}
//	fmt.Printf("Dados do User: %v\n", results["User"])
//
// Exemplo de Token Service:
// Demonstra como obter um token usando a configuração STS.
//
//	ts := api.NewTokenService()
//	ts.Configurations["my_auth"] = api.TokenConfig{...} // Configuração real
//
//	token, err := ts.GetToken("my_auth")
//	if err != nil {
//		// Tratar erro
//	}
//	fmt.Println("Token Obtido:", *token)
//
// Configuração:
// O pipeline é configurado com um slice de `APIConfig`. O TokenService
// é configurado alimentando o mapa `Configurations`.
package api
