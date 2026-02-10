package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/raywall/fast-service-lab/pkg/engine"
	"github.com/raywall/fast-service-lab/pkg/transport"
)

var (
	configPath string
	// Variáveis injetáveis para mocking
	serverStarter = transport.StartHTTPServer
	lambdaStarter = lambda.Start
)

func init() {
	// 1. Captura (Mantida no init conforme solicitado)
	configPath = os.Getenv("CONFIG_FILE_PATH")
}

func main() {
	// A validação ocorre aqui para não quebrar os testes unitários
	if configPath == "" {
		log.Fatalln("FATAL: Falha ao carregar o arquivo de roteirização")
	}

	if err := run(context.Background(), configPath); err != nil {
		log.Fatalf("FATAL: %v", err)
	}
}

// run contém a lógica principal testável
func run(ctx context.Context, cfgPath string) error {
	// 2. Carrega Configuração (Loader)
	loader := engine.NewUniversalLoader()
	cfg, err := loader.Load(ctx, cfgPath)
	if err != nil {
		return err
	}

	// 3. Inicializa Engine (Boot Time)
	svcEngine, err := engine.NewServiceEngine(cfg, cfgPath)
	if err != nil {
		return err
	}

	// 4. Seleciona Runtime Strategy
	switch cfg.Service.Runtime {
	case "local", "ec2", "ecs", "eks":
		return serverStarter(svcEngine)
	case "lambda":
		handler := transport.NewLambdaHandler(svcEngine)
		lambdaStarter(handler.Handle)
		return nil
	default:
		log.Fatalf("Runtime desconhecido: %s", cfg.Service.Runtime)
		return nil
	}
}
