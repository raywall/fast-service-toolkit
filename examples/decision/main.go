package main

import (
	"fmt"
	"log"
	"os"
	
	"github.com/raywall/fast-service-toolkit/decision/domain"
	"github.com/raywall/fast-service-toolkit/decision/infrastructure/repository"
	"github.com/raywall/fast-service-toolkit/decision/interfaces/http"
)

var repo domain.ConfigRepository

func init() {
	var err error

	source := os.Getenv("GOD_CONFIG_SOURCE")
	if source == "" {
		source = "file"
	}

	switch source {
	case "file":
		repo, err = repository.NewConfigRepositoryFromFile(`/Users/macmini/Documents/workspace/packages/fast-service-toolkit/sample-rules.yaml`)
		if err != nil {
			log.Fatal(err)
		}
	case "db":
		dbRepo, err := repository.NewDBRepository()
		if err != nil {
			log.Fatal(err)
		}
		repo = dbRepo
	default:
		log.Fatalf("GOD_CONFIG_SOURCE inválido: %s", source)
	}
}

func main() {
	r := http.SetupRouter(repo)

	if repo.GetConfig().Service.Port == 0 {
		repo.GetConfig().Service.Port = 8080
	}
	err := r.Run(fmt.Sprintf(":%d", repo.GetConfig().Service.Port))
	if err != nil {
		log.Fatalf("Erro ao iniciar o servidor: %v\n", err)
	}
}