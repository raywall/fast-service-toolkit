package main

import (
	"log"
	"sync"

	"github.com/raywall/fast-service-toolkit/tools/emulator/config"
)

// Injetável para testes
var serverStarter = func(s *config.ServerConfig) {
	s.Start()
}

func main() {
	if err := run("cmd/emulator/config.json"); err != nil {
		log.Fatalln(err)
	}
}

// run contém a lógica de orquestração
func run(configPath string) error {
	var cfg config.Config
	if err := cfg.LoadFromFile(configPath); err != nil {
		return err
	}

	var wg sync.WaitGroup
	for _, server := range []config.ServerConfig(cfg) {
		wg.Add(1)
		go func(s config.ServerConfig) {
			defer wg.Done()
			serverStarter(&s)
		}(server)
	}
	// Em modo de teste, talvez não queiramos esperar para sempre
	// mas na main real, precisamos.
	// O teste vai mockar serverStarter para não bloquear, então o wg.Wait termina.
	wg.Wait()
	return nil
}
