package transport

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mocks ---

type MockSQSClient struct {
	mock.Mock
}

func (m *MockSQSClient) ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sqs.ReceiveMessageOutput), args.Error(1)
}

func (m *MockSQSClient) DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	args := m.Called(ctx, params)
	return nil, args.Error(1)
}

// MockEngineReloader Thread-Safe
type MockEngineReloader struct {
	mu       sync.Mutex
	Reloaded bool
	Err      error
}

func (m *MockEngineReloader) Reload() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Reloaded = true
	return m.Err
}

// WasReloaded Helper para ler o estado de forma segura no teste
func (m *MockEngineReloader) WasReloaded() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Reloaded
}

// --- Tests ---

func TestSQSReloader_Integration(t *testing.T) {
	// Setup
	mockSQS := new(MockSQSClient)
	mockReloader := &MockEngineReloader{}

	// Configuração do comportamento do Mock SQS
	// 1ª chamada: Retorna uma mensagem de reload
	mockSQS.On("ReceiveMessage", mock.Anything, mock.Anything).Return(&sqs.ReceiveMessageOutput{
		Messages: []types.Message{
			{
				Body:          stringPtr(`{"action":"reload"}`),
				ReceiptHandle: stringPtr("handle_123"),
			},
		},
	}, nil).Once()

	// 2ª chamada em diante: Retorna vazio para evitar loop infinito
	mockSQS.On("ReceiveMessage", mock.Anything, mock.Anything).Return(&sqs.ReceiveMessageOutput{
		Messages: []types.Message{},
	}, nil).Maybe()

	mockSQS.On("DeleteMessage", mock.Anything, mock.Anything).Return(nil, nil)

	// Inicializa o Reloader usando o construtor da struct (CORREÇÃO AQUI)
	// Substitui a chamada direta a monitorQueue
	reloader := NewSQSReloader(mockSQS, "https://sqs.us-east-1.amazonaws.com/123/reload-queue", mockReloader)

	ctx, cancel := context.WithCancel(context.Background())

	// Executa o método Start em goroutine
	go reloader.Start(ctx)

	// Aguarda processamento
	time.Sleep(100 * time.Millisecond)

	// Para o loop
	cancel()
	time.Sleep(50 * time.Millisecond)

	// Asserts
	assert.True(t, mockReloader.WasReloaded(), "Engine deveria ter sido recarregada")

	mockSQS.AssertCalled(t, "DeleteMessage", mock.Anything, &sqs.DeleteMessageInput{
		QueueUrl:      stringPtr("https://sqs.us-east-1.amazonaws.com/123/reload-queue"),
		ReceiptHandle: stringPtr("handle_123"),
	})
}

func stringPtr(s string) *string {
	return &s
}
