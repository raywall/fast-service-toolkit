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
	ErrNotFound   = errors.New("not found")
	ErrBadRequest = errors.New("bad request")
)

// TokenConfig define a configuração de um serviço STS
type TokenConfig struct {
	GrantType    string
	ClientID     string
	ClientSecret string
	Host         string
	Httpmethod   string
}

// TokenService define os serviços para geração de token STS
type TokenService struct {
	Configurations map[string]TokenConfig
	client         *http.Client
}

// NewTokenService cria um novo serviço de token STS
func NewTokenService() *TokenService {
	return &TokenService{
		Configurations: make(map[string]TokenConfig),
		client:         &http.Client{},
	}
}

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

	req, err := http.NewRequest(token.Httpmethod, token.Host, strings.NewReader(encodedData))
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
