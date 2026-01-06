package domain

// Steps segmenta a pipeline
type Steps struct {
	Input []Step `yaml:"input"`
	Processing []Step `yaml:"processing"`
	Output []Step `yaml:"output"`
}

// Step representa uma etapa da pipeline
type Step struct {
	Type string `yaml:"type"`
	Expr string `yaml:"expr,omitempty"` // default
	OnFail OnFail `yaml:"on_fail,omitempty"`
	Datadog DatadogStep `yaml:"datadog,omitempty"`
	Msg string `yaml:"msg"`
	Fields map[string]interface{} `yaml:"fields"` // para output, suporte nested
}

// DatadogStep para métricas em steps
type DatadogStep struct {
	Metric string `yaml:"metric"`
	Action string `yaml:"action"`
	Value float64 `yaml:"value"`
	Tags map[string]string `yaml:"tags"`
}