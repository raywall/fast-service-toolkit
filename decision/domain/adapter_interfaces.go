package domain

import (
	"context"

	"google.golang.org/protobuf/types/know/structpb"
)

type CELAdapterInterface interface {
	EvalBool(expr string, input, output *structpb.Struct) (bool, error)
	EvalValue(expr string, input, output *structpb.Struct) (interface{}, error)
}

type APIAdapterInterface interface {
	CellAPI(ctx context.Context, source *Source, input *structpb.Struct) (map[string]interface{}, error)
}

type DatadogAdapterInterface interface {
	Incr(metric string, value float64, tags map[string]string)
}

type LogAdapterInterface interface {
	Log(msg string, input *structpb.Struct)
}