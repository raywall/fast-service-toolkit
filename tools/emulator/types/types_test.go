package types

import (
	"encoding/json"
	"testing"
)

func TestTypes_Marshalling(t *testing.T) {
	resp := Response{
		Status: 200,
		Body:   map[string]string{"foo": "bar"},
	}

	bytes, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Erro marshal: %v", err)
	}

	expected := `{"status":200,"body":{"foo":"bar"}}`
	if string(bytes) != expected {
		t.Errorf("JSON incorreto. Esperado %s, recebido %s", expected, string(bytes))
	}
}
