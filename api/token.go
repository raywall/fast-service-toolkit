package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
)

var (
	// ErrNotFound é retornado quando um recurso (configuração de token) não é encontrado.
	ErrNotFound = errors.New("not found")
	// ErrBadRequest é retornado para erros de requisição inválida.
	ErrBadRequest = errors.New("bad request")
)

// TokenConfig define a configuração necessária para interagir com um serviço STS
// (Security Token Service, como OAuth2/OpenID Connect).
type TokenConfig struct {
	// GrantType é o tipo de concessão de token (ex: "client_credentials").
	GrantType string
	// ClientID é o ID do cliente registrado no STS.
	ClientID string
	// ClientSecret é o segredo do cliente para autenticação.
	ClientSecret string
	// Host é a URL do endpoint de token.
	Host string
	// HttpMethod é o método HTTP (geralmente POST) para a requisição de token.
	HttpMethod string
}

// TokenService define os serviços para geração e gerenciamento de tokens STS.
//
// É o principal ponto de interação para autenticação em APIs externas.
type TokenService struct {
	// Configurations armazena as configurações de diferentes serviços STS por ID.
	Configurations map[string]TokenConfig
	// client é o cliente HTTP usado para fazer as requisições.
	client *http.Client
}

// NewTokenService cria uma nova instância do TokenService.
//
// Retorna:
//
//	*TokenService: A instância inicializada.
//
// Exemplo:
//
//	ts := NewTokenService()
func NewTokenService() *TokenService {
	return &TokenService{
		Configurations: make(map[string]TokenConfig),
		client:         &http.Client{},
	}
}

// GetToken realiza uma requisição ao serviço STS para obter um token de acesso.
//
// A requisição é montada com os dados application/x-www-form-urlencoded
// a partir da TokenConfig associada ao `tokenId`.
//
// Parâmetros:
//
//	tokenId: O identificador da configuração de token a ser usada.
//
// Retorna:
//
//	*string: O token de acesso (access_token) como um ponteiro para string, se obtido.
//	error: Um erro se a configuração não for encontrada ou a requisição falhar.
//
// Exemplo:
//
//	token, err := ts.GetToken("api_oauth")
//
// Erros:
//   - ErrNotFound: Se o `tokenId` não estiver presente em `Configurations`.
//   - Erros de rede ou I/O.
//   - Erros de decodificação JSON ou falha ao encontrar "access_token" na resposta.
func (t *TokenService) GetToken(tokenId string) (*string, error) {
	token, exists := t.Configurations[tokenId]
	if !exists {
		return nil, ErrNotFound
	}

	formData := url.Values{}
	formData.Set("grant_type", token.GrantType)
	formData.Set("client_secret", token.ClientSecret)
	formData.Set("client_id", token.ClientID)
	encodedData := formData.Encode()

	req, err := http.NewRequest(token.HttpMethod, token.Host, strings.NewReader(encodedData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response map[string]interface{}
	err = json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&response)
	if err != nil {
		return nil, err
	}
	if access_token, ok := response["access_token"].(string); ok {
		return &access_token, nil
	}

	return nil, ErrNotFound
}
