# fast-service-lab (fast-service-toolkit)

O **fast-service-toolkit** é uma engine em Go para construção de **Nanoserviços REST** e **GraphQL Mesh** de alta performance via configuração *low-code* (YAML).

O framework elimina a necessidade de escrever *boilerplate* para infraestrutura, concorrência e integrações, permitindo foco total na regra de negócio via expressões **CEL (Common Expression Language)**.

---

## Os dois pilares do toolkit

O toolkit opera em dois modos distintos, que podem coexistir no mesmo serviço:

### 1. Nanoserviços REST & API Gateway

Pipeline linear de processamento:
1.  **Input:** Recebe JSON.
2.  **Enrichment:** Busca dados em paralelo (Redis, Dynamo, HTTP, etc.).
3.  **Processing:** Aplica regras de negócio e transformações.
4.  **Output:** Retorna JSON formatado.

### 2. GraphQL mesh & federation

Motor de resolução de grafos:
1.  **Schema Definition:** Define Tipos e Campos no YAML.
2.  **Resolvers Declarativos:** Cada campo pode ter uma fonte de dados (`source`) diferente.
3.  **Execution:** O engine resolve os dados de forma assíncrona e hierárquica.

---

## Guia de configuração universal (data sources)

O coração do toolkit é o pacote `enrichment`. As fontes de dados abaixo podem ser utilizadas tanto na seção `middlewares` (REST) quanto na propriedade `source` de um campo GraphQL.

### 1. AWS Systems Manager (parameter store)

Busca configurações ou segredos na AWS.
* **Uso:** Configurações dinâmicas, feature flags, chaves de API.

```yaml
type: "aws_parameter_store"
params:
  region: "us-east-1"
  path: "/prod/service/payment/gateway_key"
  with_decryption: true # Se for SecureString

```

### 2. AWS Secrets Manager

Recupera segredos encriptados. Se o segredo for um JSON, o toolkit faz o parse automático.

* **Uso:** Credenciais de banco de dados, chaves privadas.

```yaml
type: "aws_secrets_manager"
params:
  region: "us-east-1"
  secret_id: "prod/db/postgresql_creds"

```

### 3. AWS DynamoDB

Realiza um `GetItem` otimizado.

* **Uso:** Busca de perfil de usuário, catálogo de produtos, sessão.

```yaml
type: "aws_dynamodb"
params:
  region: "us-east-1"
  table: "UsersTable"
  # Mapeamento da Chave Primária (PK)
  key:
    # No GraphQL use 'args.id' ou 'source.id'
    # No REST use 'input.user_id'
    UserId: "args.id" 

```

### 4. AWS S3 (arquivos)

Baixa e faz parse de arquivos. Suporta cache em memória (dependendo da implementação do loader).

* **Formatos:** `json`, `yaml`, `csv`, `text`.

```yaml
type: "aws_s3"
params:
  region: "us-east-1"
  bucket: "my-config-bucket"
  key: "business_rules/pricing_table.json"
  format: "json" # Realiza o Unmarshal automático para objeto

```

### 5. Redis (elasticache)

Busca dados em cache de baixa latência.

* **Uso:** Rate limiting, sessão, cache de API.

```yaml
type: "aws_redis"
params:
  host: "my-redis.cluster.us-east-1.rds.amazonaws.com:6379"
  command: "GET" # ou HGETALL
  key: "'session:' + string(input.session_id)"

```

### 6. REST (HTTP Client)

Realiza chamadas HTTP para outros serviços. Suporta injeção de Headers e Body.

```yaml
type: "rest"
params:
  method: "POST"
  url: "[https://api.parceiro.com/v1/analise](https://api.parceiro.com/v1/analise)"
  timeout: "2s" # Opcional
  body:
    documento: "input.cpf"
    valor: "input.amount"
headers:
  Content-Type: "application/json"
  Authorization: "'Bearer ' + env.API_TOKEN"

```

### 7. GraphQL client

Atua como um *client* consumindo outra API GraphQL externa.

* **Uso:** Federation, agregar dados de outros subgrafos.

```yaml
type: "graphql"
params:
  endpoint: "[https://api.github.com/graphql](https://api.github.com/graphql)"
  # A Query pode ser parametrizada
  query: |
    query($login: String!) {
      user(login: $login) {
        name
        repositories { totalCount }
      }
    }
  variables:
    login: "input.username"
headers:
  Authorization: "'Bearer ' + env.GITHUB_TOKEN"

```

### 8. Fixed (estático)

Retorna dados fixos definidos no próprio YAML.

* **Uso:** Mocks, tabelas de-para simples, valores padrão.

```yaml
type: "fixed"
params:
  tax_rate: 0.15
  category: "standard"
  features: ["read", "write"]

```

---

## Configuração do serviço (service API)

Esta seção define como expor endpoints REST e processar regras.

### Estrutura do YAML (`service`)

```yaml
service:
  name: "credit-service"
  runtime: "local" # ou lambda, ecs
  port: 8080
  route: "/analyze"
  timeout: "500ms"
  on_timeout: { code: 504, msg: "Timeout processing credit" }
  logging: { enabled: true, level: "debug" }

```

### Pipeline de execução

1. **Middlewares (Enrichment):** Executa fontes de dados em paralelo.
2. **Input (Validation):** Valida o payload de entrada.
3. **Processing (Transformation):** Executa lógica condicional.
4. **Output:** Monta a resposta.

#### Exemplo de pipeline completo

```yaml
middlewares:
  - id: "fetch_data"
    type: "enrichment"
    config:
      strategy: "parallel"
      sources:
        - name: "serasa"
          type: "rest"
          params: { method: "GET", url: "..." }
        - name: "db_interno"
          type: "aws_dynamodb"
          params: { table: "CreditHistory", ... }

steps:
  input:
    validations:
      - id: "check_amount"
        expr: "input.amount > 0"
        on_fail: { code: 400, msg: "Valor inválido" }

  processing:
    transformations:
      - name: "calc_score"
        condition: "detection.serasa.score > 700"
        value: "0.05" # Taxa de Juros Baixa
        else_value: "0.15" # Taxa de Juros Alta
        target: "vars.interest_rate"

  output:
    status_code: 200
    body:
      approved: "true"
      rate: "vars.interest_rate"

```

---

## Configuração GraphQL mesh

Esta seção define como criar um servidor GraphQL que agrega as fontes de dados acima.

### Definição de tipos e resolvers

O conceito chave é: **Cada campo tem um `source**`. Se o `source` não for definido, o engine assume que o campo já existe no objeto pai.

#### Exemplo: Tipo complexo com múltiplas fontes

```yaml
graphql:
  enabled: true
  route: "/graphql"

  types:
    UserProfile:
      fields:
        # Campo vindo do DynamoDB (Principal)
        id: { type: "ID" }
        name: { type: "String" }
        
        # Campo calculado (Fixed)
        account_type:
          type: "String"
          source:
            type: "fixed"
            params: { value: "premium" }

        # Campo aninhado vindo de API REST (Child Resolver)
        # Usa 'source.id' (ID do UserProfile carregado acima) para buscar pedidos
        recent_orders:
          type: "[Order]"
          source:
            type: "rest"
            params:
              method: "GET"
              url: "'[https://api.store.com/orders?userId=](https://api.store.com/orders?userId=)' + string(source.id)"

    Order:
      fields:
        order_id: { type: "ID" }
        total: { type: "Float" }

  query:
    getUser:
      type: "UserProfile"
      args:
        id: "ID"
      source:
        type: "aws_dynamodb"
        params:
          table: "Users"
          key: { pk: "args.id" }

```

---

## Contexto CEL (variáveis)

Onde você pode usar as variáveis nas expressões CEL (`expr`, `url`, `condition`):

| Contexto | REST (Service) | GraphQL (Mesh) | Descrição |
| --- | --- | --- | --- |
| `input` | ✅ | ❌ | JSON Body da requisição HTTP. |
| `args` | ❌ | ✅ | Argumentos da Query GraphQL (ex: `id`). |
| `source` | ❌ | ✅ | Objeto pai (usado em field resolvers aninhados). |
| `detection` | ✅ | ❌ | Dados carregados pelos middlewares. |
| `vars` | ✅ | ❌ | Variáveis calculadas no step `processing`. |
| `env` | ✅ | ✅ | Variáveis de ambiente. |
| `auth` | ✅ | ✅ | Tokens do Auth Provider. |

---

## Autenticação (auth provider)

Para serviços que precisam consumir APIs seguras (OAuth2), o toolkit gerencia o ciclo de vida do token.

```yaml
middlewares:
  - id: "auth_service_x"
    type: "auth_provider"
    config:
      provider: "oauth2"
      token_url: "[https://auth.provider.com/token](https://auth.provider.com/token)"
      client_id: "..."
      client_secret: "..."
      output_var: "token_x" # Disponível como 'auth.token_x'

```

---

## Estrutura do projeto

```text
[github.com/raywall/fast-service-lab](https://github.com/raywall/fast-service-lab)
├── cmd/server          # Entrypoint da aplicação
├── examples/           # Exemplos completos (01 a 06)
├── pkg/
│   ├── config          # Contrato das Structs YAML
│   ├── engine          # Service Engine & GraphQL Engine
│   ├── enrichment      # Implementação dos Data Sources (S3, Dynamo, REST...)
│   ├── rules           # Motor CEL (Lógica)
│   └── transport       # Servidor HTTP
└── README.md
```



### remover
```yaml
middlewares:
  - id: "auth_servico_interno"
    type: "auth_provider"
    config:
      provider: "oauth2"
      token_url: "https://seu-sts.com/oauth/token" # Endpoint que aceita o POST
      client_id: "meu-app-id"
      client_secret: "meu-segredo-super-secreto"  # Pode usar injeção aqui via envloader!
      scope: "read write"
      output_var: "token_sts" # Ficará disponível como auth.token_sts
```