package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/raywall/fast-service-lab/pkg/engine"
)

func main() {
	validateCmd := flag.NewFlagSet("validate", flag.ExitOnError)
	filePtr := validateCmd.String("file", "", "Caminho do arquivo YAML ou S3/DynamoDB URI")

	if len(os.Args) < 2 {
		fmt.Println("Comandos esperados: validate")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "validate":
		validateCmd.Parse(os.Args[2:])
		if *filePtr == "" {
			fmt.Println("Erro: flag -file é obrigatória")
			os.Exit(1)
		}
		runValidate(*filePtr)
	default:
		fmt.Println("Comando desconhecido")
		os.Exit(1)
	}
}

func runValidate(path string) {
	fmt.Printf("🔍 Analisando configuração: %s ...\n", path)

	// 1. Load (Validação Estrutural)
	loader := engine.NewUniversalLoader()
	cfg, err := loader.Load(context.Background(), path)
	if err != nil {
		fmt.Printf("❌ Erro de Carregamento/Estrutura:\n%v\n", err)
		os.Exit(1)
	}

	// 2. Analyze (Validação Lógica/Semântica)
	report, err := engine.Analyze(cfg)
	if err != nil {
		fmt.Printf("❌ Erro interno do analisador: %v\n", err)
		os.Exit(1)
	}

	if !report.Valid {
		fmt.Println("❌ A configuração contém erros lógicos:")
		for _, e := range report.Errors {
			fmt.Printf(" - %s\n", e)
		}
		os.Exit(1) // Falha no CI
	}

	// Output JSON para integração com Frontend (Angular)
	if os.Getenv("OUTPUT_FORMAT") == "json" {
		jsonOutput, _ := json.Marshal(report)
		fmt.Println(string(jsonOutput))
	} else {
		fmt.Println("✅ Configuração Válida e Pronta para Deploy!")
	}
}
