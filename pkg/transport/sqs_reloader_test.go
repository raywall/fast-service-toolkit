package transport

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/rs/zerolog"
)

// Mocks
type MockSQS struct {
	ReceiveFunc func() (*sqs.ReceiveMessageOutput, error)
}

func (m *MockSQS) ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	return m.ReceiveFunc()
}

func (m *MockSQS) DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	return &sqs.DeleteMessageOutput{}, nil
}

type MockEngineReloader struct {
	ReloadCalled bool
}

func (m *MockEngineReloader) Reload() error {
	m.ReloadCalled = true
	return nil
}

func TestSQSReloader_Integration(t *testing.T) {
	// Setup
	msgCh := make(chan *sqs.ReceiveMessageOutput, 1)
	mockSQS := &MockSQS{
		ReceiveFunc: func() (*sqs.ReceiveMessageOutput, error) {
			// Simula espera da fila ou retorno imediato
			select {
			case msg := <-msgCh:
				return msg, nil
			case <-time.After(10 * time.Millisecond):
				return &sqs.ReceiveMessageOutput{}, nil
			}
		},
	}

	mockReloader := &MockEngineReloader{}
	ctx, cancel := context.WithCancel(context.Background())
	logger := zerolog.Nop()

	// Inicia o monitor em goroutine
	go monitorQueue(ctx, mockSQS, "https://sqs.fake", mockReloader, logger)

	// Cenário: Envia mensagem simulada
	msg := "update"
	msgCh <- &sqs.ReceiveMessageOutput{
		Messages: []types.Message{{Body: &msg, ReceiptHandle: &msg}},
	}

	// Aguarda processamento (assíncrono)
	time.Sleep(100 * time.Millisecond)
	cancel() // Para o loop

	if !mockReloader.ReloadCalled {
		t.Error("Reload não foi chamado após receber mensagem SQS")
	}
}
