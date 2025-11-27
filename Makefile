.PHONY: tidy test test-coverage run dynamodb-start dynamodb-stop clean
.DEFAULT: tidy

# Variáveis para o container DynamoDB Local
DYNAMODB_CONTAINER_NAME := dynamodb-local
DYNAMODB_IMAGE := amazon/dynamodb-local:latest
DYNAMODB_PORT := 8000

tidy:
	@gofmt -w .
	@go mod tidy

test:
	@go test ./dyndb/...
	@go test -cover ./dyndb/...

test-coverage:
	@echo "Gerando relatório de cobertura em HTML..."
	@go test -coverprofile=coverage.out ./dyndb/...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Relatório de cobertura gerado: coverage.html"
	@open coverage.html || echo "Abra o arquivo coverage.html no seu navegador"

# Inicia o container do DynamoDB Local
dynamodb-start:
	@echo "Iniciando DynamoDB Local..."
	@if docker ps -a --format "table {{.Names}}" | grep -q "${DYNAMODB_CONTAINER_NAME}"; then \
		echo "Container ${DYNAMODB_CONTAINER_NAME} já existe. Reiniciando..."; \
		docker stop ${DYNAMODB_CONTAINER_NAME} >/dev/null 2>&1 || true; \
		docker rm ${DYNAMODB_CONTAINER_NAME} >/dev/null 2>&1 || true; \
	fi
	@docker run -d \
		--name ${DYNAMODB_CONTAINER_NAME} \
		-p ${DYNAMODB_PORT}:8000 \
		${DYNAMODB_IMAGE}
	@echo "Aguardando DynamoDB Local inicializar..."
	@sleep 5
	@echo "DynamoDB Local rodando em http://localhost:${DYNAMODB_PORT}"

# Para e remove o container do DynamoDB Local
dynamodb-stop:
	@echo "Parando DynamoDB Local..."
	@docker stop ${DYNAMODB_CONTAINER_NAME} >/dev/null 2>&1 || true
	@docker rm ${DYNAMODB_CONTAINER_NAME} >/dev/null 2>&1 || true
	@echo "DynamoDB Local parado e removido"

# Status do container DynamoDB
dynamodb-status:
	@if docker ps --format "table {{.Names}}" | grep -q "${DYNAMODB_CONTAINER_NAME}"; then \
		echo "DynamoDB Local está rodando"; \
	else \
		echo "DynamoDB Local não está rodando"; \
	fi

# Executa a aplicação (assume que o DynamoDB está rodando)
run:
	@DYNAMODB_ENDPOINT=http://localhost:${DYNAMODB_PORT} USERS_TABLE=local-users go run examples/users/main.go

# Inicia o DynamoDB e executa a aplicação
run-with-dynamodb: dynamodb-start run

# Limpa arquivos temporários
clean:
	@rm -f coverage.out coverage.html
	@echo "Arquivos de cobertura removidos"

# Comando completo: inicia dynamodb, roda testes com cobertura, para dynamodb
test-full: dynamodb-start test-coverage dynamodb-stop

# Ajuda
help:
	@echo "Comandos disponíveis:"
	@echo "  make tidy              - Formata código e organiza dependências"
	@echo "  make test              - Executa testes"
	@echo "  make test-coverage     - Executa testes e gera relatório HTML de cobertura"
	@echo "  make dynamodb-start    - Inicia container DynamoDB Local"
	@echo "  make dynamodb-stop     - Para e remove container DynamoDB Local"
	@echo "  make dynamodb-status   - Verifica status do container"
	@echo "  make run               - Executa aplicação (assume DynamoDB rodando)"
	@echo "  make run-with-dynamodb - Inicia DynamoDB e executa aplicação"
	@echo "  make test-full         - Executa fluxo completo: inicia dynamodb, testes com cobertura, para dynamodb"
	@echo "  make clean             - Remove arquivos temporários"