package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidator_Validate(t *testing.T) {
	validator := NewValidator()

	// Helper para criar ServiceDetails válido
	validServiceDetails := ServiceDetails{
		Name:    "test-service",
		Runtime: "local",
		Port:    8080,
		Route:   "/api",
		Timeout: "5s",
		// CORRIGIDO: Preenchendo campos obrigatórios
		OnTimeout: ErrorResponse{Code: 504, Msg: "Timeout"},
		Logging:   LoggingConf{Enabled: true, Level: "info", Format: "console"},
	}

	tests := []struct {
		name    string
		cfg     *ServiceConfig
		wantErr bool
	}{
		{
			name: "Valid Config",
			cfg: &ServiceConfig{
				Version: "1.0",
				Service: validServiceDetails,
				Steps: &StepsConf{
					Output: OutputStep{
						StatusCode: 200,
						Body:       map[string]interface{}{"msg": "ok"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Missing Steps for REST",
			cfg: &ServiceConfig{
				Version: "1.0",
				Service: validServiceDetails,
				Steps:   nil,
			},
			wantErr: true,
		},
		{
			name: "Valid GraphQL Config (No Steps)",
			cfg: &ServiceConfig{
				Version: "1.0",
				Service: validServiceDetails,
				GraphQL: GraphQLConf{Enabled: true},
				Steps:   nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
