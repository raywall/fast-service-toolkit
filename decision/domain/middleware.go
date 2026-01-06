package domain

// Middleware representa uma middleware
type Middleware struct {
	Type string `yaml:"type"`
	Config MiddlewareConfig `yaml:"config"`
}

// MiddlewareConfig representa uma configuração de uma middleware
type MiddlewareConfig struct {
	Parallel bool `yaml:"parallel"`
	Sources []Source `yaml:"sources"` // para enrichment
	RPS int `yaml:"rps,omitempty"`  // campos para rate_limit
	Burst int `yaml:"burst,omitempty"`
	JWTSecret string `yaml:"jwt_secret,omitempty"` // campos para auth
	Format string `yaml:"format,omitempty"` // campos para logging
}

// Source representa fontes de enrichment
type Source struct {
	Type string `yaml:"type"`
	Endpoint string `yaml:"endpoint,omitempty"`
	Method string `yaml:"method,omitempty"`
	Query string `yaml:"query,omitempty"`
	Variables map[string]string `yaml:"variables,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty"`
	AddTo map[string]interface{} `yaml:"add_to,omitempty"`
	OnFail MiddlewareOnFail `yaml:"on_fail,omitempty"`
}

// MiddlewareOnFail estende OnFail para middlewares
type MiddlewareOnFail struct {
	OnFail
	Action string `yaml:"action,omitempty"` // fail,skip,retry
	Retries int `yaml:"retries,omitempty"`
}