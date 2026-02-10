package config

import "time"

// ServiceConfig representa a estrutura raiz do arquivo YAML de roteirização.
type ServiceConfig struct {
	Version     string           `yaml:"version" validate:"required"`
	Service     ServiceDetails   `yaml:"service" validate:"required"`
	Middlewares []MiddlewareConf `yaml:"middlewares" validate:"dive"`
	Steps       *StepsConf       `yaml:"steps"` // Ponteiro para ser opcional no GraphQL
	GraphQL     GraphQLConf      `yaml:"graphql"`
}

// ServiceDetails contém os metadados e configurações de runtime do serviço.
type ServiceDetails struct {
	Name      string        `yaml:"name" validate:"required,hostname_rfc1123"`
	Runtime   string        `yaml:"runtime" validate:"required,oneof=local lambda ecs eks ec2"`
	Type      string        `yaml:"type"`
	Port      int           `yaml:"port" validate:"required_if=Runtime local"` // Obrigatório apenas se local
	Route     string        `yaml:"route" validate:"required,startswith=/"`
	Timeout   string        `yaml:"timeout" validate:"required"` // Ex: "500ms", "2s"
	OnTimeout ErrorResponse `yaml:"on_timeout"`
	Logging   LoggingConf   `yaml:"logging"`
	Metrics   MetricsConf   `yaml:"metrics"`
}

type GraphQLConf struct {
	Enabled        bool                `yaml:"enabled"`
	Route          string              `yaml:"route"`
	SQSReloadQueue string              `json:"sqs_reload_queue" yaml:"sqs_reload_queue"`
	Types          map[string]GQLType  `yaml:"types"`
	Query          map[string]GQLField `yaml:"query"`
	Mutation       map[string]GQLField `yaml:"mutation"`
}

type GQLType struct {
	Description string              `yaml:"description"`
	Fields      map[string]GQLField `yaml:"fields"`
}

type GQLField struct {
	Type        string                  `yaml:"type"`
	Description string                  `yaml:"description"`
	Args        map[string]string       `yaml:"args"`
	Source      *EnrichmentSourceConfig `yaml:"source"`
}

type EnrichmentSourceConfig struct {
	Type    string                 `yaml:"type"`
	Params  map[string]interface{} `yaml:"params"`
	Headers map[string]string      `yaml:"headers"`
}

type ErrorResponse struct {
	Code int    `yaml:"code" validate:"gte=400,lt=600"`
	Msg  string `yaml:"msg" validate:"required"`
}

type LoggingConf struct {
	Enabled bool   `yaml:"enabled"`
	Level   string `yaml:"level" validate:"oneof=debug info warn error"`
	Format  string `yaml:"format" validate:"oneof=json console"`
}

type MetricsConf struct {
	Datadog DatadogConf `yaml:"datadog"`
}

type DatadogConf struct {
	Enabled           bool                     `yaml:"enabled" env:"DD_ENABLED"`
	Addr              string                   `yaml:"addr" env:"DD_AGENT_HOST" validate:"required_if=Enabled true"`
	Namespace         string                   `yaml:"namespace"`
	CustomDefinitions []CustomMetricDefinition `yaml:"custom_definitions" validate:"dive"`
}

type CustomMetricDefinition struct {
	ID   string `yaml:"id" validate:"required"`
	Name string `yaml:"name" validate:"required"`
	Type string `yaml:"type" validate:"oneof=count gauge histogram"`
}

type MiddlewareConf struct {
	Type   string                 `yaml:"type" validate:"required,oneof=rate_limit auth_provider enrichment"`
	ID     string                 `yaml:"id" validate:"required"`
	Config map[string]interface{} `yaml:"config" validate:"required"`
}

type StepsConf struct {
	Input      InputStep      `yaml:"input"`
	Processing ProcessingStep `yaml:"processing"`
	Output     OutputStep     `yaml:"output"`
}

type InputStep struct {
	Validations []ValidationRule `yaml:"validations" validate:"dive"`
}

type ProcessingStep struct {
	Validations     []ValidationRule     `yaml:"validations" validate:"dive"`
	Transformations []TransformationRule `yaml:"transformations" validate:"dive"`
}

type OutputStep struct {
	StatusCode  int                      `yaml:"status_code" validate:"gte=200,lt=600"`
	Body        map[string]interface{}   `yaml:"body" validate:"required"` // Mantido como MAP para o analyzer funcionar
	Headers     map[string]string        `yaml:"headers"`
	Target      TargetConf               `yaml:"target"`
	Validations []ValidationRule         `yaml:"validations" validate:"dive"`
	Metrics     []MetricRegistrationRule `yaml:"metrics" validate:"dive"`
}

type TargetConf struct {
	URL     string `yaml:"url"`
	Method  string `yaml:"method"`
	Timeout string `yaml:"timeout"`
}

type ValidationRule struct {
	ID     string        `yaml:"id" validate:"required"`
	Expr   string        `yaml:"expr" validate:"required"`
	OnFail ErrorResponse `yaml:"on_fail" validate:"required"`
}

type TransformationRule struct {
	Name      string `yaml:"name" validate:"required"`
	Condition string `yaml:"condition" validate:"required"`
	Value     string `yaml:"value" validate:"required"`
	ElseValue string `yaml:"else_value"`
	Target    string `yaml:"target" validate:"required"`
}

type MetricRegistrationRule struct {
	MetricID string            `yaml:"metric_id" validate:"required"`
	Value    string            `yaml:"value" validate:"required"`
	Tags     map[string]string `yaml:"tags"`
}

func (s ServiceDetails) GetTimeout() time.Duration {
	d, err := time.ParseDuration(s.Timeout)
	if err != nil {
		return 30 * time.Second
	}
	return d
}
