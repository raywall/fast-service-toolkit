package transport

import (
	"context"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/raywall/fast-service-lab/pkg/config"
	"github.com/raywall/fast-service-lab/pkg/engine"
	"github.com/stretchr/testify/assert"
)

func TestLambdaHandler_REST(t *testing.T) {
	// Config
	cfg := &config.ServiceConfig{
		Service: config.ServiceDetails{
			Name:    "lambda-test",
			Timeout: "1s",
		},
		// CORRIGIDO: Ponteiro &StepsConf
		Steps: &config.StepsConf{
			Output: config.OutputStep{
				StatusCode: 200,
				Body: map[string]interface{}{
					"msg": "hello lambda",
				},
			},
		},
	}

	eng, _ := engine.NewServiceEngine(cfg, "memory")
	handler := NewLambdaHandler(eng)

	// Evento API Gateway
	req := events.APIGatewayProxyRequest{
		Body: `{}`,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	resp, err := handler.Handle(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Contains(t, resp.Body, "hello lambda")
}
