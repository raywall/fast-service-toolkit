package graphql

import (
	"testing"

	"github.com/raywall/fast-service-lab/pkg/config"
	"github.com/raywall/fast-service-lab/pkg/rules"
)

func TestMakeResolver_ParamResolution(t *testing.T) {
	rm, _ := rules.NewRuleManager()

	// Definição do Campo com CEL nos params
	fieldDef := config.GQLField{
		Type: "String",
		Source: &config.EnrichmentSourceConfig{
			Type: "fixed",
			Params: map[string]interface{}{
				"dynamic_id": "args.id", // Vem do argumento da query
				"static_val": "teste",
				"calc_val":   "'PREFIX_' + source.code", // Vem do objeto pai
			},
		},
	}

	// OBS: Não precisamos chamar makeResolver aqui, pois vamos testar
	// a lógica de resolução de parâmetros (resolveParams) diretamente abaixo.

	// Simula execução do GraphQL
	mockSource := map[string]interface{}{"code": "123"}
	mockArgs := map[string]interface{}{"id": 999}

	celCtx := map[string]interface{}{
		"args":   mockArgs,
		"source": mockSource,
	}

	// Testamos a função interna que o Resolver usa
	resolved, err := resolveParams(fieldDef.Source.Params, celCtx, rm)
	if err != nil {
		t.Fatalf("Erro na resolução: %v", err)
	}

	// Validações
	if toString(resolved["dynamic_id"]) != "999" {
		t.Errorf("Arg 'id' não resolvido. Recebido: %v", resolved["dynamic_id"])
	}
	if toString(resolved["calc_val"]) != "PREFIX_123" {
		t.Errorf("Source 'code' não resolvido. Recebido: %v", resolved["calc_val"])
	}
	if toString(resolved["static_val"]) != "teste" {
		t.Errorf("Valor estático alterado. Recebido: %v", resolved["static_val"])
	}
}

// Teste do Header Resolution
func TestResolveHeaders(t *testing.T) {
	rm, _ := rules.NewRuleManager()

	rawHeaders := map[string]string{
		"Authorization": "'Bearer ' + auth.token",
		"X-Static":      "Fixo",
	}

	ctx := map[string]interface{}{
		"auth": map[string]interface{}{"token": "abc-123"},
	}

	res, err := resolveHeaders(rawHeaders, ctx, rm)
	if err != nil {
		t.Fatalf("Erro: %v", err)
	}

	if res["Authorization"] != "Bearer abc-123" {
		t.Errorf("Auth header incorreto: %s", res["Authorization"])
	}
	if res["X-Static"] != "Fixo" {
		t.Errorf("Header estático incorreto: %s", res["X-Static"])
	}
}
