package repository

import (
	"fmt"
	"log"
	"os"

	"github.com/raywall/fast-service-toolkit/decision/domain"
	"gopkg.in/yaml.v2"
)

// FileRepository carrega config de arquivo
type FileRepository struct {
	filePath string
	config *domain.Config
}

// NewFileRepository instancia um novo repositorio do tipo arquivo
func NewFileRepository(filePath string) (*FileRepository, error) {
	repo := &FileRepository{
		filePath: filePath,
		config: &domain.Config{},
	}

	config, err := repo.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("falha ao carregar configurações: %v", err)
	}
	repo.config = config

	return repo, nil
}

// LoadConfig carrega de YAML.
func (r *FileRepository) LoadConfig() (*domain. Config, error) {
	if r.filePath == "" {
		r.filePath = "config.yaml"
	}

	// Ler arquivo diretamente
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return nil, err
	}

	var config domain.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		log.Printf("ERROR: YAML unmarshal falled: %v\n", err)
		return nil, err
	}

	r.config = &config
	return &config, nil
}

// GetConfig recupera as configurações do serviço
func (r *FileRepository) GetConfig() *domain.Config {
	return r.config
}