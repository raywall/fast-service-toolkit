package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// APIPipelineInterface realiza a abstração da estrutura APIPipeline.
//
// Esta interface define o contrato mínimo para a execução do pipeline.
type APIPipelineInterface interface {
	// Execute inicia a execução do pipeline de APIs.
	Execute(ctx context.Context, input interface{}) (map[string]interface{}, error)
}

// APIConfig define a configuração necessária para uma chamada de API.
//
// É o bloco de construção fundamental do pipeline, especificando como e
// quando uma API deve ser chamada.
type APIConfig struct {
	// Name é o identificador único da API no pipeline.
	Name string
	// Required, se true, qualquer falha desta API cancela todo o pipeline
	// e retorna um erro 422 (circuit break).
	Required bool
	// Dependencies lista os nomes das APIs (APIConfig.Name) das quais esta API depende.
	Dependencies []string
	// Parameters contém os detalhes de como a chamada HTTP será montada.
	Parameters APIParameters
}

// APIParameters define os parâmetros e argumentos para a chamada da API.
type APIParameters struct {
	// AccessToken é o token de autenticação (se aplicável).
	AccessToken *string
	// HttpMethod é o método HTTP (GET, POST, PUT, etc.).
	HttpMethod string
	// Host é a URL completa para a chamada da API.
	Host string
	// Body é o corpo da requisição, serializado para JSON.
	Body map[string]interface{}
	// Headers são os cabeçalhos HTTP adicionais para a requisição.
	Headers map[string]string
}

// APIPipeline gerencia a execução paralela de APIs com dependências.
//
// Ele coordena o fluxo de execução, coleta resultados e trata erros
// de forma concorrente.
type APIPipeline struct {
	// apis é a lista de todas as APIs a serem executadas.
	apis []APIConfig
	// results armazena os dados de resposta das APIs bem-sucedidas.
	results map[string]interface{}
	// errors armazena os erros retornados pelas APIs.
	errors map[string]error
	// mu protege o acesso concorrente a results e errors.
	mu sync.RWMutex
	// resultChan é um canal para receber resultados das goroutines de API.
	resultChan chan APIResult
	// client é o cliente HTTP usado para fazer as requisições.
	client *http.Client
}

// APIResult encapsula o resultado de uma chamada de API, seja sucesso ou falha.
type APIResult struct {
	// Name é o nome da API que retornou o resultado.
	Name string
	// Data contém o corpo da resposta HTTP (JSON decodificado).
	Data interface{}
	// Error contém o erro da chamada, se houver.
	Error error
}

// NewAPIPipeline cria uma nova instância do APIPipeline.
//
// Parâmetros:
//   apis: O slice de APIConfig que define todo o fluxo do pipeline.
//
// Retorna:
//   APIPipelineInterface: A instância configurada do pipeline, pronta para ser executada.
//
// Exemplo:
//   p := NewAPIPipeline([]APIConfig{...})
func NewAPIPipeline(apis []APIConfig) APIPipelineInterface {
	return &APIPipeline{
		apis:       apis,
		results:    make(map[string]interface{}),
		errors:     make(map[string]error),
		resultChan: make(chan APIResult, len(apis)),
		client:     &http.Client{},
	}
}

// Execute executa todas as APIs respeitando dependências e concorrência.
//
// O método gerencia a lógica de agendamento (iniciar APIs sem dependências
// e, em seguida, iniciar APIs dependentes à medida que os resultados
// são coletados).
//
// Parâmetros:
//   ctx: Contexto para controle de timeout ou cancelamento.
//   input: Payload de entrada (atualmente não usado na lógica de dependência).
//
// Retorna:
//   map[string]interface{}: Um mapa onde a chave é o nome da API e o valor
//     é a resposta HTTP decodificada.
//   error: Retorna um erro se o contexto for cancelado ou se uma API
//     obrigatória (`Required`) falhar (erro 422).
//
// Erros:
//   - "contexto cancelado": Quando o `ctx.Done()` é disparado.
//   - "erro 422: API obrigatória '%s' falhou": Se `APIConfig.Required` for true e a chamada falhar.
func (p *APIPipeline) Execute(ctx context.Context, input interface{}) (map[string]interface{}, error) {
	start := time.Now()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Inicializa estruturas
	p.mu.Lock()
	p.results = make(map[string]interface{})
	p.errors = make(map[string]error)
	p.mu.Unlock()

	resultChan := make(chan APIResult, len(p.apis))
	completed := make(map[string]bool)
	started := make(map[string]bool)
	totalAPIs := len(p.apis)
	completedCount := 0
	var mu sync.Mutex

	// Função para executar uma API
	executeAPI := func(api APIConfig) {
		// Obtém dependências
		var deps map[string]interface{}
		p.mu.RLock()
		if len(api.Dependencies) > 0 {
			deps = make(map[string]interface{})
			for _, dep := range api.Dependencies {
				if data, ok := p.results[dep]; ok {
					deps[dep] = data
				}
			}
		}
		p.mu.RUnlock()

		data, err := api.call(ctx, deps, p.client)

		select {
		case resultChan <- APIResult{Name: api.Name, Data: data, Error: err}:
		case <-ctx.Done():
		}
	}

	// Inicia APIs sem dependências
	mu.Lock()
	for _, api := range p.apis {
		if len(api.Dependencies) == 0 {
			started[api.Name] = true
			go executeAPI(api)
		}
	}
	mu.Unlock()

	// Loop principal de processamento
	for completedCount < totalAPIs {
		select {
		case result := <-resultChan:
			// Processa resultado
			p.mu.Lock()
			if result.Error != nil {
				p.errors[result.Name] = result.Error
				slog.Debug("API failed", "name", result.Name, "error", result.Error)

				// Se é obrigatória, cancela
				for _, api := range p.apis {
					if api.Name == result.Name && api.Required {
						cancel()
					}
				}
			} else {
				p.results[result.Name] = result.Data
				slog.Debug("API completed", "name", result.Name)
			}
			p.mu.Unlock()

			// Marca como completada e incrementa contador
			mu.Lock()
			completed[result.Name] = true
			completedCount++

			// Verifica e inicia APIs dependentes que estão prontas
			for _, api := range p.apis {
				// Já foi iniciada? Pula
				if started[api.Name] {
					continue
				}

				// Sem dependências? Não deveria estar aqui
				if len(api.Dependencies) == 0 {
					continue
				}

				// Verifica se todas as dependências foram completadas
				allDepsReady := true
				for _, dep := range api.Dependencies {
					if !completed[dep] {
						allDepsReady = false
						break
					}
				}

				// Se todas as dependências estão prontas, inicia
				if allDepsReady {
					started[api.Name] = true
					go executeAPI(api)
				}
			}
			mu.Unlock()

		case <-ctx.Done():
			// Contexto cancelado, retorna erro apropriado
			return nil, fmt.Errorf("contexto cancelado durante execução do pipeline")
		}
	}

	// Verifica APIs obrigatórias
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, api := range p.apis {
		if api.Required {
			if err, exists := p.errors[api.Name]; exists {
				return nil, fmt.Errorf("erro 422: API obrigatória '%s' falhou: %w", api.Name, err)
			}
			if _, exists := p.results[api.Name]; !exists {
				return nil, fmt.Errorf("erro 422: API obrigatória '%s' não retornou dados", api.Name)
			}
		}
	}

	slog.Debug("Pipeline executada!", "duration", time.Since(start).Milliseconds())
	return p.results, nil
}

// call realiza a chamada HTTP real para uma API definida em APIConfig.
//
// É responsável por serializar o corpo da requisição, montar os cabeçalhos
// e decodificar a resposta JSON.
//
// Parâmetros:
//   ctx: Contexto de requisição.
//   deps: Resultados das APIs dependentes (não utilizado no call, mas passível de uso para transformar o payload).
//   client: Cliente HTTP.
//
// Retorna:
//   map[string]interface{}: O corpo da resposta decodificado.
//   error: Erros de rede, I/O, ou falha de marshalling/unmarshalling JSON.
//
// Erros:
//   - "error marshalling to JSON": Falha ao serializar o corpo para JSON.
//   - "error creating request": Falha ao montar o objeto *http.Request.
//   - "error making request": Erro de rede na chamada http.Client.Do().
//   - "error reading response body": Falha ao ler o corpo da resposta.
//   - "error decoding response": Falha ao decodificar o corpo da resposta como JSON.
func (p *APIPipeline) executeAPI(ctx context.Context, api APIConfig, deps map[string]interface{}, wg *sync.WaitGroup) {
	defer wg.Done()

	data, err := api.call(ctx, deps, p.client)

	select {
	case <-ctx.Done():
		// Contexto cancelado, ignora o resultado
		return
	default:
		// Tenta enviar o resultado, mas não bloqueia se o canal estiver fechado
		select {
		case p.resultChan <- APIResult{Name: api.Name, Data: data, Error: err}:
		case <-ctx.Done():
		}
	}
}

// tryExecuteDependents tenta executar APIs que dependem de outras já completadas
func (p *APIPipeline) tryExecuteDependents(ctx context.Context, wg *sync.WaitGroup,
	executed map[string]bool, executedMu *sync.Mutex) {

	executedMu.Lock()
	defer executedMu.Unlock()

	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, api := range p.apis {
		// Se já foi executada, pula
		if executed[api.Name] {
			continue
		}

		// Verifica se todas as dependências foram satisfeitas
		canExecute := true
		for _, dep := range api.Dependencies {
			if !executed[dep] {
				canExecute = false
				break
			}
		}

		if canExecute && len(api.Dependencies) > 0 {
			// Marca como executada ANTES de iniciar para evitar execução duplicada
			executed[api.Name] = true

			// Obtém dependências
			deps := make(map[string]interface{})
			for _, dep := range api.Dependencies {
				if data, ok := p.results[dep]; ok {
					deps[dep] = data
				}
			}

			wg.Add(1)
			go p.executeAPI(ctx, api, deps, wg)
		}
	}
}

// call define a assinatura de uma chamada de API
func (c *APIConfig) call(ctx context.Context, deps map[string]interface{}, client *http.Client) (map[string]interface{}, error) {
	// registra inicio do processamento
	start := time.Now()

	var payload io.Reader = nil
	if c.Parameters.Body != nil {
		slice, err := json.Marshal(c.Parameters.Body)
		if err != nil {
			return nil, fmt.Errorf("error marshalling to JSON: %w", err)
		}
		payload = bytes.NewReader(slice)
	}

	req, err := http.NewRequest(c.Parameters.HttpMethod, c.Parameters.Host, payload)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	for key, value := range c.Parameters.Headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var response map[string]interface{}
	err = json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	// calcula o tempo de processamento
	duration := time.Since(start)
	slog.Debug("API executada!", "duration", duration.Milliseconds())

	return response, nil
}
