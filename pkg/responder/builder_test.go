package responder

import (
	"encoding/json"
	"testing"

	"github.com/raywall/fast-service-toolkit/pkg/config"
	"github.com/raywall/fast-service-toolkit/pkg/rules"
	"github.com/stretchr/testify/assert"
)

func TestResponseBuilder_Build(t *testing.T) {
	rm, _ := rules.NewRuleManager()

	cfg := config.OutputStep{
		StatusCode: 201,
		Body: map[string]interface{}{
			"id":       "${input.id}",
			"mensagem": "Processado com sucesso", // O normalizer agora lida bem com isso
			"ativo":    true,
		},
		Headers: map[string]string{
			"X-User-ID": "${input.id}",
		},
	}

	builder, err := NewResponseBuilder(cfg, rm)
	assert.NoError(t, err)

	ctx := map[string]interface{}{
		"input": map[string]interface{}{
			"id": 123,
		},
	}

	code, bodyBytes, headers, err := builder.Build(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 201, code)
	assert.Equal(t, "123", headers["X-User-ID"])

	var bodyMap map[string]interface{}
	json.Unmarshal(bodyBytes, &bodyMap)
	assert.Equal(t, 123.0, bodyMap["id"]) // JSON numbers are floats
	assert.Equal(t, "Processado com sucesso", bodyMap["mensagem"])
	assert.Equal(t, true, bodyMap["ativo"])
}
