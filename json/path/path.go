package path

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Extrator é responsável por extrair valores de estruturas JSON usando JSONPath
type Extrator struct {
	data map[string]interface{}
}

// NovoExtrator cria uma nova instância do extrator a partir de bytes JSON
func NovoExtrator(jsonBytes []byte) (*Extrator, error) {
	var data map[string]interface{}

	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return nil, fmt.Errorf("erro ao fazer parse do JSON: %w", err)
	}

	return &Extrator{data: data}, nil
}

// NovoExtrotorFromMap cria uma nova instância do extrator a partir de um map
func NovoExtrotorFromMap(data map[string]interface{}) *Extrator {
	return &Extrator{data: data}
}

// Extrair extrai um valor do JSON usando um caminho no formato JSONPath
// Exemplos de caminhos válidos:
//   - "nome" -> retorna valor direto
//   - "dados_profissionais" -> retorna objeto completo
//   - "dados_profissionais.empregador" -> navega em objetos aninhados
//   - "cursos" -> retorna array completo
//   - "cursos[0]" -> retorna primeiro elemento do array
//   - "cursos[1].nome" -> retorna campo de um elemento do array
func (e *Extrator) Extrair(caminho string) (interface{}, error) {
	// Remove espaços em branco
	caminho = strings.TrimSpace(caminho)

	// Se o caminho estiver vazio, retorna o objeto completo
	if caminho == "" {
		return e.data, nil
	}

	// Divide o caminho em partes
	partes := parseCaminho(caminho)

	// Navega pela estrutura
	var atual interface{} = e.data

	for i, parte := range partes {
		// Verifica se é acesso a array
		if parte.isArray {
			atual = navegarArray(atual, parte, partes, i)
			if atual == nil {
				return nil, fmt.Errorf("erro ao acessar array no caminho: %s", caminho)
			}
		} else {
			// Navegação normal em objetos
			m, ok := atual.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("esperado objeto no caminho '%s', mas encontrou %T", construirCaminhoAteAqui(partes, i), atual)
			}

			valor, existe := m[parte.campo]
			if !existe {
				return nil, fmt.Errorf("campo '%s' não encontrado no caminho '%s'", parte.campo, construirCaminhoAteAqui(partes, i+1))
			}

			atual = valor
		}
	}

	return atual, nil
}

// ExtrairString é um helper que extrai e converte para string
func (e *Extrator) ExtrairString(caminho string) (string, error) {
	valor, err := e.Extrair(caminho)
	if err != nil {
		return "", err
	}

	switch v := valor.(type) {
	case string:
		return v, nil
	case nil:
		return "", fmt.Errorf("valor é null")
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

// ExtrairInt é um helper que extrai e converte para int
func (e *Extrator) ExtrairInt(caminho string) (int, error) {
	valor, err := e.Extrair(caminho)
	if err != nil {
		return 0, err
	}

	switch v := valor.(type) {
	case float64:
		return int(v), nil
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("não foi possível converter %T para int", valor)
	}
}

// ExtrairFloat é um helper que extrai e converte para float64
func (e *Extrator) ExtrairFloat(caminho string) (float64, error) {
	valor, err := e.Extrair(caminho)
	if err != nil {
		return 0, err
	}

	switch v := valor.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("não foi possível converter %T para float64", valor)
	}
}

// ExtrairBool é um helper que extrai e converte para bool
func (e *Extrator) ExtrairBool(caminho string) (bool, error) {
	valor, err := e.Extrair(caminho)
	if err != nil {
		return false, err
	}

	switch v := valor.(type) {
	case bool:
		return v, nil
	case string:
		return strconv.ParseBool(v)
	default:
		return false, fmt.Errorf("não foi possível converter %T para bool", valor)
	}
}

// ExtrairArray é um helper que extrai e retorna um slice
func (e *Extrator) ExtrairArray(caminho string) ([]interface{}, error) {
	valor, err := e.Extrair(caminho)
	if err != nil {
		return nil, err
	}

	arr, ok := valor.([]interface{})
	if !ok {
		return nil, fmt.Errorf("valor em '%s' não é um array, encontrado %T", caminho, valor)
	}

	return arr, nil
}

// ExtrairObjeto é um helper que extrai e retorna um map
func (e *Extrator) ExtrairObjeto(caminho string) (map[string]interface{}, error) {
	valor, err := e.Extrair(caminho)
	if err != nil {
		return nil, err
	}

	obj, ok := valor.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("valor em '%s' não é um objeto, encontrado %T", caminho, valor)
	}

	return obj, nil
}

// Existe verifica se um caminho existe no JSON
func (e *Extrator) Existe(caminho string) bool {
	_, err := e.Extrair(caminho)
	return err == nil
}

// --- Estruturas e funções auxiliares ---

// parteCaminho representa uma parte do caminho JSONPath
type parteCaminho struct {
	campo   string
	isArray bool
	indice  int
}

// parseCaminho converte uma string de caminho em partes estruturadas
func parseCaminho(caminho string) []parteCaminho {
	var partes []parteCaminho

	// Divide por pontos, mas preserva arrays
	segmentos := strings.Split(caminho, ".")

	for _, segmento := range segmentos {
		if segmento == "" {
			continue
		}

		// Verifica se tem acesso a array
		if strings.Contains(segmento, "[") {
			// Exemplo: "cursos[0]" ou apenas "[0]"
			abreBracket := strings.Index(segmento, "[")
			fechaBracket := strings.Index(segmento, "]")

			if fechaBracket == -1 {
				// Bracket não fechado, trata como campo normal
				partes = append(partes, parteCaminho{campo: segmento})
				continue
			}

			// Extrai o nome do campo (se existir)
			nomeCampo := segmento[:abreBracket]
			if nomeCampo != "" {
				partes = append(partes, parteCaminho{campo: nomeCampo})
			}

			// Extrai o índice
			indiceStr := segmento[abreBracket+1 : fechaBracket]
			indice, err := strconv.Atoi(indiceStr)
			if err != nil {
				// Índice inválido, ignora
				continue
			}

			partes = append(partes, parteCaminho{
				isArray: true,
				indice:  indice,
			})

			// Verifica se há algo após o bracket
			restoSegmento := segmento[fechaBracket+1:]
			if restoSegmento != "" {
				// Recursivamente processa o resto
				partesResto := parseCaminho(restoSegmento)
				partes = append(partes, partesResto...)
			}
		} else {
			partes = append(partes, parteCaminho{campo: segmento})
		}
	}

	return partes
}

// navegarArray navega em um array e retorna o elemento no índice especificado
func navegarArray(atual interface{}, parte parteCaminho, todasPartes []parteCaminho, indiceAtual int) interface{} {
	arr, ok := atual.([]interface{})
	if !ok {
		return nil
	}

	if parte.indice < 0 || parte.indice >= len(arr) {
		return nil
	}

	return arr[parte.indice]
}

// construirCaminhoAteAqui reconstrói o caminho até um determinado índice (para mensagens de erro)
func construirCaminhoAteAqui(partes []parteCaminho, ate int) string {
	var builder strings.Builder

	for i := 0; i < ate && i < len(partes); i++ {
		if i > 0 && !partes[i].isArray {
			builder.WriteString(".")
		}

		if partes[i].isArray {
			builder.WriteString(fmt.Sprintf("[%d]", partes[i].indice))
		} else {
			builder.WriteString(partes[i].campo)
		}
	}

	return builder.String()
}

// ExtrairMultiplos extrai múltiplos caminhos de uma só vez
func (e *Extrator) ExtrairMultiplos(caminhos ...string) (map[string]interface{}, error) {
	resultado := make(map[string]interface{})

	for _, caminho := range caminhos {
		valor, err := e.Extrair(caminho)
		if err != nil {
			return nil, fmt.Errorf("erro ao extrair '%s': %w", caminho, err)
		}
		resultado[caminho] = valor
	}

	return resultado, nil
}

// ToJSON converte o valor extraído de volta para JSON
func ToJSON(valor interface{}) ([]byte, error) {
	return json.Marshal(valor)
}

// ToJSONIndent converte o valor extraído para JSON formatado
func ToJSONIndent(valor interface{}) ([]byte, error) {
	return json.MarshalIndent(valor, "", "  ")
}
