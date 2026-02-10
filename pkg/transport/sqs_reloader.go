package transport

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/rs/zerolog"
)

// SQSClient define a interface necessária para o reloader (permite Mocking)
type SQSClient interface {
	ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
}

// Interface para desacoplar o transport do engine
type Reloader interface {
	Reload() error
}

func StartSQSReloader(ctx context.Context, queueUrl string, reloader Reloader, logger zerolog.Logger) {
	if queueUrl == "" {
		return
	}

	// Carrega configuração real apenas se não estivermos em teste (ou injetamos via func param, mas aqui simplificamos)
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Erro ao carregar AWS SDK para SQS")
		return
	}
	client := sqs.NewFromConfig(cfg)

	// Delega para a versão interna testável
	go monitorQueue(ctx, client, queueUrl, reloader, logger)
}

// monitorQueue contém a lógica do loop, aceitando a interface
func monitorQueue(ctx context.Context, client SQSClient, queueUrl string, reloader Reloader, logger zerolog.Logger) {
	logger.Info().Str("queue", queueUrl).Msg("📡 Monitorando fila SQS para Hot Reload")

	for {
		select {
		case <-ctx.Done():
			return
		default:
			out, err := client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
				QueueUrl:            aws.String(queueUrl),
				MaxNumberOfMessages: 1,
				WaitTimeSeconds:     20, // Long polling
			})

			if err != nil {
				// Backoff simples em caso de erro de rede
				time.Sleep(5 * time.Second)
				continue
			}

			if len(out.Messages) > 0 {
				logger.Info().Msg("🔔 Evento de alteração recebido via SQS!")

				// Dispara o Reload no Engine
				if err := reloader.Reload(); err != nil {
					logger.Error().Err(err).Msg("❌ Falha crítica no Reload")
				}

				// Apaga a mensagem para não processar de novo
				_, _ = client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
					QueueUrl:      aws.String(queueUrl),
					ReceiptHandle: out.Messages[0].ReceiptHandle,
				})
			}
		}
	}
}
