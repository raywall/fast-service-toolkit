package engine

import (
	"context"

	"github.com/raywall/fast-service-toolkit/pkg/config"
)

// Loader é responsável por carregar e decodificar a configuração do serviço.
// Ele abstrai a origem do arquivo (Sistema de arquivos, S3, URL, etc).
type Loader interface {
	// Load lê a configuração a partir de uma origem e retorna a struct validada.
	Load(ctx context.Context, source string) (*config.ServiceConfig, error)
}

// Engine é o cérebro do framework.
// É responsável por "compilar" a configuração em tempo de inicialização (Boot Time),
// preparando os middlewares, parsers CEL e clientes de conexão.
type Engine interface {
	// Init inicializa o motor com a configuração fornecida.
	// Retorna um Executor pronto para processar requisições ou erro.
	Init(cfg *config.ServiceConfig) (Executor, error)
}

// Executor é a interface de tempo de execução (Runtime).
// Ela deve ser extremamente performática e thread-safe, pois será chamada
// concorrentemente por cada requisição HTTP/Evento.
type Executor interface {
	// Execute processa uma única requisição seguindo o roteiro do serviço.
	// Recebe o payload bruto (bytes) e retorna a resposta processada (bytes) ou erro.
	Execute(ctx context.Context, payload []byte) (int, []byte, map[string]string, error)

	// Shutdown realiza o encerramento gracioso de recursos (fechar conexões, flush de logs).
	Shutdown(ctx context.Context) error
}
