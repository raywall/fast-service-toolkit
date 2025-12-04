package tools

import (
	"encoding/json"
	"testing"
)

var jsonTeste = []byte(`{
	"nome": "jose",
	"idade": 17,
	"ativo": true,
	"salario": 5000.50,
	"dados_profissionais": {
		"empregador": "itau",
		"data_admissao": "2025-01-01",
		"cargo": {
			"titulo": "Desenvolvedor",
			"nivel": "Senior"
		}
	},
	"cursos": [
		{ "nome": "informatica", "conclusao": 2025 },
		{ "nome": "digitacao", "conclusao": 2024 }
	],
	"tags": ["golang", "backend", "api"]
}`)

func TestNovoExtrator(t *testing.T) {
	extrator, err := NovoExtrator(jsonTeste)
	if err != nil {
		t.Fatalf("Erro ao criar extrator: %v", err)
	}
	
	if extrator == nil {
		t.Fatal("Extrator não deveria ser nil")
	}
}

func TestNovoExtrotorJSONInvalido(t *testing.T) {
	jsonInvalido := []byte(`{nome: jose}`)
	
	_, err := NovoExtrator(jsonInvalido)
	if err == nil {
		t.Fatal("Deveria retornar erro para JSON inválido")
	}
}

func TestExtrairCampoSimples(t *testing.T) {
	extrator, _ := NovoExtrator(jsonTeste)
	
	valor, err := extrator.Extrair("nome")
	if err != nil {
		t.Fatalf("Erro ao extrair nome: %v", err)
	}
	
	if valor != "jose" {
		t.Errorf("Esperado 'jose', obtido '%v'", valor)
	}
}

func TestExtrairNumero(t *testing.T) {
	extrator, _ := NovoExtrator(jsonTeste)
	
	valor, err := extrator.Extrair("idade")
	if err != nil {
		t.Fatalf("Erro ao extrair idade: %v", err)
	}
	
	if valor != float64(17) {
		t.Errorf("Esperado 17, obtido %v", valor)
	}
}

func TestExtrairBooleano(t *testing.T) {
	extrator, _ := NovoExtrator(jsonTeste)
	
	valor, err := extrator.Extrair("ativo")
	if err != nil {
		t.Fatalf("Erro ao extrair ativo: %v", err)
	}
	
	if valor != true {
		t.Errorf("Esperado true, obtido %v", valor)
	}
}

func TestExtrairObjetoCompleto(t *testing.T) {
	extrator, _ := NovoExtrator(jsonTeste)
	
	valor, err := extrator.Extrair("dados_profissionais")
	if err != nil {
		t.Fatalf("Erro ao extrair dados_profissionais: %v", err)
	}
	
	obj, ok := valor.(map[string]interface{})
	if !ok {
		t.Fatalf("Esperado map[string]interface{}, obtido %T", valor)
	}
	
	if obj["empregador"] != "itau" {
		t.Errorf("Esperado empregador 'itau', obtido '%v'", obj["empregador"])
	}
}

func TestExtrairCampoAninhado(t *testing.T) {
	extrator, _ := NovoExtrator(jsonTeste)
	
	valor, err := extrator.Extrair("dados_profissionais.empregador")
	if err != nil {
		t.Fatalf("Erro ao extrair empregador: %v", err)
	}
	
	if valor != "itau" {
		t.Errorf("Esperado 'itau', obtido '%v'", valor)
	}
}

func TestExtrairCampoTresNiveis(t *testing.T) {
	extrator, _ := NovoExtrator(jsonTeste)
	
	valor, err := extrator.Extrair("dados_profissionais.cargo.titulo")
	if err != nil {
		t.Fatalf("Erro ao extrair titulo: %v", err)
	}
	
	if valor != "Desenvolvedor" {
		t.Errorf("Esperado 'Desenvolvedor', obtido '%v'", valor)
	}
}

func TestExtrairArrayCompleto(t *testing.T) {
	extrator, _ := NovoExtrator(jsonTeste)
	
	valor, err := extrator.Extrair("cursos")
	if err != nil {
		t.Fatalf("Erro ao extrair cursos: %v", err)
	}
	
	arr, ok := valor.([]interface{})
	if !ok {
		t.Fatalf("Esperado []interface{}, obtido %T", valor)
	}
	
	if len(arr) != 2 {
		t.Errorf("Esperado array com 2 elementos, obtido %d", len(arr))
	}
}

func TestExtrairElementoArray(t *testing.T) {
	extrator, _ := NovoExtrator(jsonTeste)
	
	valor, err := extrator.Extrair("cursos[0]")
	if err != nil {
		t.Fatalf("Erro ao extrair cursos[0]: %v", err)
	}
	
	obj, ok := valor.(map[string]interface{})
	if !ok {
		t.Fatalf("Esperado map[string]interface{}, obtido %T", valor)
	}
	
	if obj["nome"] != "informatica" {
		t.Errorf("Esperado 'informatica', obtido '%v'", obj["nome"])
	}
}

func TestExtrairCampoDeElementoArray(t *testing.T) {
	extrator, _ := NovoExtrator(jsonTeste)
	
	valor, err := extrator.Extrair("cursos[1].nome")
	if err != nil {
		t.Fatalf("Erro ao extrair cursos[1].nome: %v", err)
	}
	
	if valor != "digitacao" {
		t.Errorf("Esperado 'digitacao', obtido '%v'", valor)
	}
}

func TestExtrairArraySimples(t *testing.T) {
	extrator, _ := NovoExtrator(jsonTeste)
	
	valor, err := extrator.Extrair("tags[0]")
	if err != nil {
		t.Fatalf("Erro ao extrair tags[0]: %v", err)
	}
	
	if valor != "golang" {
		t.Errorf("Esperado 'golang', obtido '%v'", valor)
	}
}

func TestExtrairCampoInexistente(t *testing.T) {
	extrator, _ := NovoExtrator(jsonTeste)
	
	_, err := extrator.Extrair("campo_inexistente")
	if err == nil {
		t.Fatal("Deveria retornar erro para campo inexistente")
	}
}

func TestExtrairIndiceForaDoLimite(t *testing.T) {
	extrator, _ := NovoExtrator(jsonTeste)
	
	_, err := extrator.Extrair("cursos[10]")
	if err == nil {
		t.Fatal("Deveria retornar erro para índice fora do limite")
	}
}

func TestExtrairIndiceNegativo(t *testing.T) {
	extrator, _ := NovoExtrator(jsonTeste)
	
	_, err := extrator.Extrair("cursos[-1]")
	if err == nil {
		t.Fatal("Deveria retornar erro para índice negativo")
	}
}

func TestExtrairString(t *testing.T) {
	extrator, _ := NovoExtrator(jsonTeste)
	
	valor, err := extrator.ExtrairString("nome")
	if err != nil {
		t.Fatalf("Erro ao extrair string: %v", err)
	}
	
	if valor != "jose" {
		t.Errorf("Esperado 'jose', obtido '%s'", valor)
	}
}

func TestExtrairInt(t *testing.T) {
	extrator, _ := NovoExtrator(jsonTeste)
	
	valor, err := extrator.ExtrairInt("idade")
	if err != nil {
		t.Fatalf("Erro ao extrair int: %v", err)
	}
	
	if valor != 17 {
		t.Errorf("Esperado 17, obtido %d", valor)
	}
}

func TestExtrairFloat(t *testing.T) {
	extrator, _ := NovoExtrator(jsonTeste)
	
	valor, err := extrator.ExtrairFloat("salario")
	if err != nil {
		t.Fatalf("Erro ao extrair float: %v", err)
	}
	
	if valor != 5000.50 {
		t.Errorf("Esperado 5000.50, obtido %.2f", valor)
	}
}

func TestExtrairBool(t *testing.T) {
	extrator, _ := NovoExtrator(jsonTeste)
	
	valor, err := extrator.ExtrairBool("ativo")
	if err != nil {
		t.Fatalf("Erro ao extrair bool: %v", err)
	}
	
	if valor != true {
		t.Errorf("Esperado true, obtido %v", valor)
	}
}

func TestExtrairArray(t *testing.T) {
	extrator, _ := NovoExtrator(jsonTeste)
	
	valor, err := extrator.ExtrairArray("cursos")
	if err != nil {
		t.Fatalf("Erro ao extrair array: %v", err)
	}
	
	if len(valor) != 2 {
		t.Errorf("Esperado 2 elementos, obtido %d", len(valor))
	}
}

func TestExtrairObjeto(t *testing.T) {
	extrator, _ := NovoExtrator(jsonTeste)
	
	valor, err := extrator.ExtrairObjeto("dados_profissionais")
	if err != nil {
		t.Fatalf("Erro ao extrair objeto: %v", err)
	}
	
	if len(valor) == 0 {
		t.Error("Objeto não deveria estar vazio")
	}
}

func TestExiste(t *testing.T) {
	extrator, _ := NovoExtrator(jsonTeste)
	
	casos := []struct {
		caminho  string
		esperado bool
	}{
		{"nome", true},
		{"idade", true},
		{"dados_profissionais.empregador", true},
		{"cursos[0].nome", true},
		{"campo_inexistente", false},
		{"cursos[10]", false},
		{"dados_profissionais.campo_falso", false},
	}
	
	for _, caso := range casos {
		resultado := extrator.Existe(caso.caminho)
		if resultado != caso.esperado {
			t.Errorf("Existe('%s'): esperado %v, obtido %v", caso.caminho, caso.esperado, resultado)
		}
	}
}

func TestExtrairMultiplos(t *testing.T) {
	extrator, _ := NovoExtrator(jsonTeste)
	
	valores, err := extrator.ExtrairMultiplos("nome", "idade", "dados_profissionais.empregador")
	if err != nil {
		t.Fatalf("Erro ao extrair múltiplos: %v", err)
	}
	
	if len(valores) != 3 {
		t.Errorf("Esperado 3 valores, obtido %d", len(valores))
	}
	
	if valores["nome"] != "jose" {
		t.Errorf("Valor incorreto para 'nome'")
	}
}

func TestToJSON(t *testing.T) {
	extrator, _ := NovoExtrator(jsonTeste)
	
	valor, _ := extrator.Extrair("dados_profissionais")
	jsonBytes, err := ToJSON(valor)
	
	if err != nil {
		t.Fatalf("Erro ao converter para JSON: %v", err)
	}
	
	var resultado map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &resultado); err != nil {
		t.Fatalf("JSON gerado é inválido: %v", err)
	}
}

func TestCaminhosComplexos(t *testing.T) {
	jsonComplexo := []byte(`{
		"empresa": {
			"departamentos": [
				{
					"nome": "TI",
					"funcionarios": [
						{"nome": "João", "idade": 30},
						{"nome": "Maria", "idade": 25}
					]
				},
				{
					"nome": "RH",
					"funcionarios": [
						{"nome": "Pedro", "idade": 35}
					]
				}
			]
		}
	}`)
	
	extrator, _ := NovoExtrator(jsonComplexo)
	
	// Navegar em estrutura profunda
	valor, err := extrator.Extrair("empresa.departamentos[0].funcionarios[1].nome")
	if err != nil {
		t.Fatalf("Erro ao extrair caminho complexo: %v", err)
	}
	
	if valor != "Maria" {
		t.Errorf("Esperado 'Maria', obtido '%v'", valor)
	}
}

func BenchmarkExtrairCampoSimples(b *testing.B) {
	extrator, _ := NovoExtrator(jsonTeste)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extrator.Extrair("nome")
	}
}

func BenchmarkExtrairCampoAninhado(b *testing.B) {
	extrator, _ := NovoExtrator(jsonTeste)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extrator.Extrair("dados_profissionais.empregador")
	}
}

func BenchmarkExtrairArray(b *testing.B) {
	extrator, _ := NovoExtrator(jsonTeste)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extrator.Extrair("cursos[1].nome")
	}
}

func BenchmarkExtrairMultiplos(b *testing.B) {
	extrator, _ := NovoExtrator(jsonTeste)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extrator.ExtrairMultiplos("nome", "idade", "dados_profissionais.empregador", "cursos[0].nome")
	}
}