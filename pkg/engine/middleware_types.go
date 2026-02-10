package engine

// Estruturas auxiliares para decodificar a configuração dos middlewares "on the fly"
type EnrichmentConfig struct {
	Strategy string             `json:"strategy"`
	Sources  []EnrichmentSource `json:"sources"`
}

type EnrichmentSource struct {
	Name    string                 `json:"name"`
	Type    string                 `json:"type"`
	Params  map[string]interface{} `json:"params"`
	Headers map[string]string      `json:"headers"`
}

type RateLimitConfig struct {
	RPS   int `json:"rps"`
	Burst int `json:"burst"`
}

// AuthConfig define a configuração para obtenção de tokens
type AuthConfig struct {
	Provider string `json:"provider"`
	TokenURL string `json:"token_url"`
	ClientID string `json:"client_id"`
	Secret   string `json:"client_secret"`
	Scope    string `json:"scope"`
}
