package adapter

import (
	"github.com/DataDog/datadog.go/statsd"
	"github.com/raywall/fast-service-toolkit/decision/domain"
)

type DatadogAdapter struct {
	client *statsd.Client
}

func NewDatadogAdapter(addr string) *DatadogAdapter {
	client, err := statsd.New(addr)
	if err != nil {
		return &DatadogAdapter{client: &statsd.Client{}} // fallback
	}
	return &DatadogAdapter{client: client}
}

func (d *DatadogAdapter) Incr(metric string, value float64, tags map[string]string) error {
	ddTags := make([]string, 0, len(tags))
	for k, v := range tags {
		ddTags = append(ddTags, k+":"+v)
	}
	return d.client.Incr(metric, ddTags, value)
}

var _ domain.DatadogAdapterInterface = (*DatadogAdapter)(nil)
