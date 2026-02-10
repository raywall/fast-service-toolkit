# Busca dinâmica: Procura arquivos que comecem com "01-", "1-", etc.
CONF_FILE = $(shell find examples -name "0$(case)-*.yaml" -o -name "$(case)-*.yaml" | head -n 1)
CONFIG_FILE_PATH = "/Users/raysouz/Downloads/user-gateway.yaml"

.PHONY: update test debug coverage run build pack graphql start
.DEFAULT: run

# 1. Executa o Server com o cenário (case) escolhido
run:
	@if [ -z "$(case)" ]; then echo "❌ Erro: Informe o número do cenário. Ex: make run case=1"; exit 1; fi
	@if [ -z "$(CONF_FILE)" ]; then echo "❌ Erro: Arquivo de configuração não encontrado para o case $(case)"; exit 1; fi
	@echo "🚀 \033[1;32mIniciando Servidor [Cenário $(case)]\033[0m"
	@echo "📂 Config: $(CONF_FILE)"
	@echo "---------------------------------------------------"
	@MY_API_TOKEN="dev-token-123" \
	 CONFIG_FILE_PATH=$(CONF_FILE) \
	 go run cmd/server/main.go

# 2. Executa o Curl específico para o cenário (case) escolhido
call:
	@if [ -z "$(case)" ]; then echo "❌ Erro: Informe o número do cenário. Ex: make call case=1"; exit 1; fi
	@echo "📞 \033[1;34mChamando API [Cenário $(case)]\033[0m"
	@echo "---------------------------------------------------"
	@# A estrutura abaixo é: "switch (variavel_make) { ... }"
	@case $(case) in \
		1) curl -i -X POST http://localhost:8080/get-user -d '{"user_id": 4}' ;; \
		2) curl -i -X POST http://localhost:8081/profile -d '{}' ;; \
		3) curl -i -X POST http://localhost:8082/secure-data -d '{}' ;; \
		4) curl -i -X POST http://localhost:8085/graphql -d '{"query": "query { getCustomer(id: \"ea0768db-b6dd-509e-8679-cf1eb5172777\") { id_pessoa, cargo, orders { total } } }"}' ;; \
		5) curl -i -X POST http://localhost:8090/api/graphql -d '{"query": "query { getUsuario(id: \"ea0768db-b6dd-509e-8679-cf1eb5172777\") { id_pessoa, cargo, ativo, scopes } }"}' ;; \
		6) curl -i -X POST http://localhost:8095/graphql -d '{"query": "query { getDashboard(userId: \"1\", docType: \"contract\") { public_posts { title }, secure_record { authenticated } } }"}' ;; \
		7) curl -i -X GET  http://localhost:8080/v1/customer/5 ;; \
		8) curl -i -X POST http://localhost:9090/graphql -H 'Content-Type: application/json' -d '{"query": "query { listPokemons(limit: 2) { count results { name url social_comments { name body } } } }" }' ;; \
		9) curl -i -X POST http://localhost:9090/transactions/authorize -H 'Content-Type: application/json' -d '{"transaction_id": "tx_987654321", "amount": 6000, "currency": "BRL"}' ;; \
		*) echo "⚠️  Curl não definido para o cenário $(case)" ;; \
		
	 esac
	@echo "\n"

server: update
	@go run cmd/emulator/main.go

update:
	@go mod tidy

test: update
	@if [ -z "$(case)" ]; then go test ./...; exit 1; fi
	@case $(case) in \
		7) CONFIG_FILE_PATH=examples/customer-aggregator.yaml go run cmd/server/main.go ;; \
		8) CONFIG_FILE_PATH=examples/poke-social.yaml go run cmd/server/main.go ;; \
		9) CONFIG_FILE_PATH=examples/risk-interceptor.yaml go run cmd/server/main.go ;; \
		*) echo "⚠️  Cenário não definido para $(case)" ;; \
	 esac
	@echo "\n"

debug: update
	@go test ./... -v

coverage: update
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o=cover.html

build: update test
	@go build -o bootstrap .

pack: build
	@zip application.zip bootstrap

start:
	@CONFIG_FILE_PATH=${CONFIG_FILE_PATH} go run cmd/server/main.go