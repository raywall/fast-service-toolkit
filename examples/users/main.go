// Copyright 2025 Raywall Malheiros de Souza
// Licensed under the Mozilla Public License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.mozilla.org/en-US/MPL/2.0/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/raywall/fast-service-toolkit/examples/users/models"
	"github.com/raywall/fast-service-toolkit/examples/users/repository"
)

func main() {
	ctx := context.Background()

	// Config AWS (local ou prod)
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal("erro AWS config:", err)
	}

	client := dynamodb.NewFromConfig(cfg)

	// === USO DO REPOSITORIO ===
	userRepo := repository.NewUserRepository(client)

	// Exemplo 1: Save um novo usuário
	newUser := &models.User{
		UserID: "user-999",
		Email:  "new@example.com",
		Name:   "Novo Usuário",
		Status: "active",
	}
	if err := userRepo.Save(ctx, newUser); err != nil {
		log.Printf("Save erro: %v", err)
	} else {
		log.Println("Usuário salvo com sucesso!")
	}

	// Exemplo 2: GetByEmail (GSI)
	user, err := userRepo.GetByEmail(ctx, "john@example.com")
	if err != nil {
		log.Printf("Usuário não encontrado: %v", err)
	} else {
		log.Printf("Usuário: %+v", user)
	}

	// Exemplo 3: ListActive com paginação
	users, nextToken, err := userRepo.ListActive(ctx, 10, "")
	if err != nil {
		log.Fatal("erro list:", err)
	}
	log.Printf("Usuários ativos: %d", len(users))
	if nextToken != "" {
		log.Printf("Próxima página: %s", nextToken)
	}

	// Exemplo 4: BatchGet
	ids := []string{"user-123", "user-456"}
	batchUsers, err := userRepo.BatchGetByIDs(ctx, ids)
	if err != nil {
		log.Printf("Batch get erro: %v", err)
	} else {
		log.Printf("Batch users: %d", len(batchUsers))
	}

	// Exemplo 5: Delete
	if err := userRepo.Delete(ctx, "user-999", "new@example.com"); err != nil {
		log.Printf("Delete erro: %v", err)
	} else {
		log.Println("Usuário deletado!")
	}
}
