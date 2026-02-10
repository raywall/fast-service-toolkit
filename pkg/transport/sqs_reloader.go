package transport

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// SQSClient define a interface necessária para o reloader (permite Mocking)
type SQSClient interface {
	ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
}

// Reloader define a interface para recarregar o engine
type Reloader interface {
	Reload() error
}

// SQSReloader gerencia o loop de verificação do SQS
type SQSReloader struct {
	client   SQSClient
	queueUrl string
	reloader Reloader
	logger   zerolog.Logger
}

// NewSQSReloader cria uma nova instância do reloader
// CORREÇÃO: Esta função estava faltando e causava o erro "undefined: NewSQSReloader"
func NewSQSReloader(client SQSClient, queueUrl string, reloader Reloader) *SQSReloader {
	return &SQSReloader{
		client:   client,
		queueUrl: queueUrl,
		reloader: reloader,
		logger:   log.With().Str("component", "sqs_reloader").Logger(),
	}
}

// Start inicia o monitoramento (bloqueante)
func (s *SQSReloader) Start(ctx context.Context) {
	if s.queueUrl == "" {
		s.logger.Warn().Msg("URL da fila SQS não configurada. Hot Reload desativado.")
		return
	}

	s.logger.Info().Str("queue", s.queueUrl).Msg("📡 Monitorando fila SQS para Hot Reload")

	for {
		select {
		case <-ctx.Done():
			s.logger.Info().Msg("Parando monitoramento SQS")
			return
		default:
			out, err := s.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
				QueueUrl:            aws.String(s.queueUrl),
				MaxNumberOfMessages: 1,
				WaitTimeSeconds:     20, // Long polling
			})

			if err != nil {
				if ctx.Err() != nil {
					return
				}
				s.logger.Error().Err(err).Msg("Erro no SQS. Retentando em 5s...")
				time.Sleep(5 * time.Second)
				continue
			}

			if len(out.Messages) > 0 {
				s.logger.Info().Msg("🔔 Evento de alteração recebido via SQS!")

				if err := s.reloader.Reload(); err != nil {
					s.logger.Error().Err(err).Msg("❌ Falha crítica no Reload")
				} else {
					s.logger.Info().Msg("✅ Hot Reload aplicado")
				}

				_, _ = s.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
					QueueUrl:      aws.String(s.queueUrl),
					ReceiptHandle: out.Messages[0].ReceiptHandle,
				})
			}
		}
	}
}
