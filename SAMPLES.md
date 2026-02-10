# Fast Service Toolkit - Exemplos de Uso

Este documento detalha três cenários arquiteturais complexos resolvidos declarativamente com o Fast Service Toolkit.

---

## 1. Customer 360º Aggregator (REST Service)

**Arquivo:** `examples/customer-aggregator.yaml`

### Problema

Precisamos de uma API unificada que receba um ID de usuário, busque seus dados cadastrais em um sistema legado (simulado), verifique seu status financeiro (score) e calcule dinamicamente se ele é elegível para um cartão "Black".

### Solução

Utilizamos o modo **REST Service** com:

1.  **Auth Provider:** Simula a obtenção de um token Machine-to-Machine para chamar APIs internas.
2.  **Enrichment Paralelo:** Busca dados do usuário e posts recentes simultaneamente.
3.  **Lógica de Negócio (CEL):**
    * Valida se o ID é numérico.
    * Calcula um "Score Interno" baseado no tamanho do nome da empresa do usuário (lógica arbitrária para demo).
    * Transforma esse score em uma categoria (GOLD/SILVER).
4.  **Output:** Agrega tudo em um JSON limpo para o Frontend.

### Script: REST Service (`examples/customer-aggregator.yaml`)

```yaml
version: '1.0'

service:
  name: customer-360
  runtime: local
  port: 9090
  route: /v1/customer/{id}
  timeout: 500ms
  type: rest

  on_timeout:
    code: 504
    msg: Gateway Timeout
  
  logging:
    level: info
    format: console
    enabled: true

middlewares:
  - id: internal_auth
    type: auth_provider
    config:
      provider: oauth2
      # Httpbin retorna um JSON qualquer, simulando um token response
      token_url: http://localhost:8082/token
      client_id: my-client
      client_secret: my-secret
      output_var: access_token
  
  # Busca dados em paralelo
  - id: fetch_data
    type: enrichment
    config:
      strategy: parallel
      stop_on_error: true
      sources:
        # Fonte 1: Dados do Usuário
        - name: user_data
          type: rest
          params:
            url: https://jsonplaceholder.typicode.com/users/${input.id}
            method: GET
            timeout: 2s

        # Fonte 2: Posts recentes (Simulando histórico financeiro)
        - name: user_posts
          type: rest
          params:
            url: https://jsonplaceholder.typicode.com/posts?userId=${input.id}
            method: GET
            timeout: 2s

steps:
  input:
    validations:
      # Garante que ID é numérico e positivo
      - id: validate_id
        expr: input.id.matches('^[0-9]+$')
        on_fail:
          code: 400
          msg: ID must be numeric

  processing:
    validations: []
    transformations:
      # Lógica de Negócio: Calcula Score baseado no tamanho do nome da empresa (Demo)
      - name: calc_score
        condition: true 
        value: detection.user_data.company.name.size() * 100
        else_value: '0'
        target: vars.score

      # Categorização do Cliente
      - name: categorize_tier
        condition: int(vars.score) > 1000
        value: "'PLATINUM'"
        else_value: "'STANDARD'"
        target: vars.tier

    metrics: []

  output:
    status_code: 200
    body:
      customer_id: ${input.id}
      profile:
        name: ${detection.user_data.name}
        email: ${detection.user_data.email}
        city: ${detection.user_data.address.city}

      financial_analysis:
        score: ${vars.score}
        tier: ${vars.tier}
        active_contracts: ${detection.user_posts.size()}

      metadata:
        processed_at: time.now()

    headers:
      X-Calculated-Tier: ${vars.tier}
      X-Internal-Token: ${auth.internal_auth.access_token}  # Demonstra uso do token gerado

    metrics: []
```

### Teste

```bash
curl -i -X GET http://localhost:9090/v1/customer/5 
```

### Resposta

```bash
HTTP/1.1 200 OK
Content-Type: application/json
X-Calculated-Tier: PLATINUM
X-Correlation-Id: db2dc7a9-af6f-42fe-888b-4d2abb077146
X-Internal-Token: mock-jwt-token-999
X-Latency-Ms: 94
Date: Mon, 09 Feb 2026 23:18:30 GMT
Content-Length: 229

{
    "customer_id": "5",
    "financial_analysis": {
        "active_contracts": 10,
        "score": 1100,
        "tier": "PLATINUM"
    },
    "metadata": {
        "processed_at": "time.now()"
    },
    "profile": {
        "city": "Roscoeview",
        "email": "Lucio_Hettinger@annie.ca",
        "name": "Chelsey Dietrich"
    }
}
```

---

## 2. Poke-Social Federation (GraphQL Mesh)

**Arquivo:** `examples/poke-social.yaml`

### O Problema

O time de Frontend precisa exibir uma lista de Pokémons, mas quer enriquecer essa lista com "comentários de treinadores" que estão armazenados em um banco SQL legado (simulado por uma API REST), e não na API oficial do Pokémon.

### A Solução

Utilizamos o modo **GraphQL Mesh** para federar dados:

1.  **Schema Unificado:** Definimos tipos `Pokemon` e `Comment`.
2.  **Resolvers Híbridos:**
    * A query principal busca dados na API GraphQL pública (`beta.pokeapi.co`).
    * O campo `comments` dentro de cada Pokémon dispara uma chamada REST para `jsonplaceholder` injetando o ID do Pokémon.
3.  **Resultado:** O cliente faz uma única query GraphQL e o Toolkit orquestra as chamadas (N+1 problem mitigado pela concorrência do Go).

### O Script: GraphQL Mesh (`examples/poke-social.yaml`)

```yaml
version: '1.0'
service:
  name: poke-social-mesh
  runtime: local
  port: 9090
  route: /graphql # Rota do Playground/API
  timeout: 5s     # Aumentei o timeout pois chamadas externas podem demorar
  type: graphql
  
  on_timeout:
    code: 504
    msg: Gateway Timeout

  logging:
    level: info
    format: console
    enabled: true

graphql:
  enabled: true
  route: /graphql

  # Definição dos Tipos (Schema)
  types:
    Pokemon:
      description: Um monstrinho de bolso
      fields:
        name:
          type: String
        url:
          type: String

        # Este campo é resolvido via REST API (Federation)
        social_comments:
          type: '[Comment]'
          description: Comentários de treinadores sobre este pokemon
          source:
            type: rest
            params:
              # Usamos uma API fake de comentários, passando o ID do pokemon como postId
              # 'source' aqui se refere ao objeto pai (Pokemon)
              method: GET
              url: https://jsonplaceholder.typicode.com/comments?postId=1
              # Nota: Na API real JSONPlaceholder, postId=1 sempre retorna os mesmos.
              # Num caso real, faríamos parsing da URL para pegar o ID real do pokemon.

    Comment:
      description: Comentário de um usuário
      fields:
        id: { type: Int }
        name: { type: String }
        body: { type: String }
        email: { type: String }

    PokemonList:
      fields:
        count: { type: Int }
        # CORRIGIDO: Aspas para indicar string "[Pokemon]"
        results: { type: "[Pokemon]" }

  # Entry Point
  query:
    listPokemons:
      type: PokemonList
      args:
        limit: Int
      source:
        type: graphql
        params:
          response_path: pokemons
          endpoint: https://graphql-pokeapi.graphcdn.app/
          query: |
            query($limit: Int) {
              pokemons(limit: $limit) {
                count
                results {
                  name
                  url
                }
              }
            }
            
          variables:
            limit: int(args.limit)
```

### Teste

```bash
curl -X POST http://localhost:9090/graphql \
     -H "Content-Type: application/json" \
     -d '{
        "query": "query { listPokemons(limit: 2) { count results { name url social_comments { name body } } } }"
     }'
```

### Resposta

```json
{
  "data": {
    "listPokemons": {
      "count": null,
      "results": [
        {
          "name": "ivysaur",
          "social_comments": [
            {
              "body": "laudantium enim quasi est quidem magnam voluptate ipsam eos\ntempora quo necessitatibus\ndolor quam autem quasi\nreiciendis et nam sapiente accusantium",
              "name": "id labore ex et quam laborum"
            },
            {
              "body": "est natus enim nihil est dolore omnis voluptatem numquam\net omnis occaecati quod ullam at\nvoluptatem error expedita pariatur\nnihil sint nostrum voluptatem reiciendis et",
              "name": "quo vero reiciendis velit similique earum"
            },
            {
              "body": "quia molestiae reprehenderit quasi aspernatur\naut expedita occaecati aliquam eveniet laudantium\nomnis quibusdam delectus saepe quia accusamus maiores nam est\ncum et ducimus et vero voluptates excepturi deleniti ratione",
              "name": "odio adipisci rerum aut animi"
            },
            {
              "body": "non et atque\noccaecati deserunt quas accusantium unde odit nobis qui voluptatem\nquia voluptas consequuntur itaque dolor\net qui rerum deleniti ut occaecati",
              "name": "alias odio sit"
            },
            {
              "body": "harum non quasi et ratione\ntempore iure ex voluptates in ratione\nharum architecto fugit inventore cupiditate\nvoluptates magni quo et",
              "name": "vero eaque aliquid doloribus et culpa"
            }
          ],
          "url": "https://pokeapi.co/api/v2/pokemon/2/"
        },
        {
          "name": "venusaur",
          "social_comments": [
            {
              "body": "laudantium enim quasi est quidem magnam voluptate ipsam eos\ntempora quo necessitatibus\ndolor quam autem quasi\nreiciendis et nam sapiente accusantium",
              "name": "id labore ex et quam laborum"
            },
            {
              "body": "est natus enim nihil est dolore omnis voluptatem numquam\net omnis occaecati quod ullam at\nvoluptatem error expedita pariatur\nnihil sint nostrum voluptatem reiciendis et",
              "name": "quo vero reiciendis velit similique earum"
            },
            {
              "body": "quia molestiae reprehenderit quasi aspernatur\naut expedita occaecati aliquam eveniet laudantium\nomnis quibusdam delectus saepe quia accusamus maiores nam est\ncum et ducimus et vero voluptates excepturi deleniti ratione",
              "name": "odio adipisci rerum aut animi"
            },
            {
              "body": "non et atque\noccaecati deserunt quas accusantium unde odit nobis qui voluptatem\nquia voluptas consequuntur itaque dolor\net qui rerum deleniti ut occaecati",
              "name": "alias odio sit"
            },
            {
              "body": "harum non quasi et ratione\ntempore iure ex voluptates in ratione\nharum architecto fugit inventore cupiditate\nvoluptates magni quo et",
              "name": "vero eaque aliquid doloribus et culpa"
            }
          ],
          "url": "https://pokeapi.co/api/v2/pokemon/3/"
        }
      ]
    }
  }
}
```

---

## 3. Legacy Banking Modernizer (Interceptor)

**Arquivo:** `examples/risk-interceptor.yaml`

### O Problema

Um Core Bancário legado (Mainframe exposto via REST) exige que toda requisição de transação contenha a geolocalização do IP do cliente e um header `X-Risk-Level`. Os apps mobile modernos não enviam isso.

### A Solução

Utilizamos o modo **Interceptor (Smart Proxy)**:

1.  **Input:** Recebe a transação crua do App.
2.  **Enrichment:** Usa a API `ip-api.com` para descobrir o país e cidade baseados no IP fictício do cliente.
3.  **Regras de Risco:**
    * Se o valor > 5000 E o país não for o de origem, marca Risco `HIGH`.
    * Caso contrário, `LOW`.

4.  **Forwarding:**
    * Injeta o objeto `geo_location` no body.
    * Injeta o header `X-Risk-Level`.
    * Encaminha para o `httpbin.org` (simulando o Core Bancário) e devolve a resposta dele.

### O Script: Interceptor Mode (`examples/risk-interceptor.yaml`)

```yaml
version: "1.0"
service:
  name: "risk-analysis-interceptor"
  runtime: "local"
  port: 8082
  route: "/transactions/authorize"
  timeout: "5s"

middlewares:
  - type: "enrichment"
    id: "geo_intelligence"
    config:
      strategy: "parallel"
      sources:
        # Simula obter dados de GeoIP. Usamos um IP fixo do Google para garantir sucesso na demo.
        # Em prod, usariamos ${input.client_ip}
        - name: "geo_data"
          type: "rest"
          params:
            url: "http://ip-api.com/json/8.8.8.8" 
            method: "GET"

steps:
  input:
    validations:
      - id: "chk_amount"
        expr: "input.amount > 0"
        on_fail: { code: 400, msg: "Invalid amount" }

  processing:
    transformations:
      # Regra de Risco: Valor alto (> 5000) OU País fora do esperado (diferente de US para este IP)
      - name: "assess_risk"
        target: "vars.risk_level"
        condition: "input.amount > 5000 || detection.geo_data.countryCode != 'US'"
        value: "'HIGH'"
        else_value: "'LOW'"

      - name: "generate_audit_id"
        target: "vars.audit_id"
        condition: "true"
        value: "'AUDIT-' + string(input.transaction_id)"

  output:
    # Construção do Payload Enriquecido para o Sistema Legado
    body:
      original_transaction:
        id: "${input.transaction_id}"
        val: "${input.amount}"
        curr: "${input.currency}"
      
      # Dados injetados pelo Interceptor
      risk_assessment:
        level: "${vars.risk_level}"
        audit_key: "${vars.audit_id}"
      
      geo_context:
        country: "${detection.geo_data.country}"
        isp: "${detection.geo_data.isp}"
        timezone: "${detection.geo_data.timezone}"

    # Headers mandatórios para o legado
    headers:
      X-Risk-Class: "${vars.risk_level}"
      X-Forwarded-By: "Fast-Service-Interceptor"

    # Forwarding (Proxy): Envia para o 'Core Bancário' (Simulado pelo HttpBin)
    target:
      url: "https://httpbin.org/post"
      method: "POST"
      timeout: "5s"
```

### Teste

```bash
curl -i -X POST http://localhost:9090/transactions/authorize \
     -H "Content-Type: application/json" \
     -d '{
        "transaction_id": "tx_987654321",
        "amount": 6000,
        "currency": "BRL"
     }'
```

### Resposta

```bash
HTTP/1.1 200 OK
Access-Control-Allow-Credentials: true
Access-Control-Allow-Origin: *
Content-Length: 1097
Content-Type: application/json
Date: Mon, 09 Feb 2026 23:47:37 GMT
Server: gunicorn/19.9.0
X-Correlation-Id: c85b9b38-d93e-4810-abd0-12f09af16340
X-Latency-Ms: 1178

{
  "args": {}, 
  "data": "{\"geo_context\":{\"country\":\"United States\",\"isp\":\"Google LLC\",\"timezone\":\"America/New_York\"},\"original_transaction\":{\"curr\":\"BRL\",\"id\":\"tx_987654321\",\"val\":6000},\"risk_assessment\":{\"audit_key\":\"AUDIT-tx_987654321\",\"level\":\"HIGH\"}}", 
  "files": {}, 
  "form": {}, 
  "headers": {
    "Accept-Encoding": "gzip", 
    "Content-Length": "229", 
    "Content-Type": "application/json", 
    "Host": "httpbin.org", 
    "User-Agent": "FastServiceToolkit/Interceptor", 
    "X-Amzn-Trace-Id": "Root=1-698a7219-12e3e97769e4d63446d74262", 
    "X-Forwarded-By": "Fast-Service-Interceptor", 
    "X-Risk-Class": "HIGH"
  }, 
  "json": {
    "geo_context": {
      "country": "United States", 
      "isp": "Google LLC", 
      "timezone": "America/New_York"
    }, 
    "original_transaction": {
      "curr": "BRL", 
      "id": "tx_987654321", 
      "val": 6000
    }, 
    "risk_assessment": {
      "audit_key": "AUDIT-tx_987654321", 
      "level": "HIGH"
    }
  }, 
  "origin": "189.121.203.11", 
  "url": "https://httpbin.org/post"
}
```