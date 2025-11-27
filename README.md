# dynamodb-quick-service: Kit de ferramentas Go para backend otimizado

O `dynamodb-quick-service` é uma biblioteca modular em Go projetada para acelerar a construção de backends robustos, especialmente aqueles que utilizam o AWS DynamoDB e necessitam de orquestração de APIs externas e **carregamento de configuração simplificado**.

Este pacote consolida três utilitários poderosos: **`envloader`** (Configuração), **`dyndb`** (Persistência) e **`api`** (Orquestração).

## Instalação

```bash
go get [github.com/raywall/dynamodb-quick-service](https://github.com/raywall/dynamodb-quick-service)
```

## Estrutura Modular

A biblioteca é dividida em três sub-pacotes principais, cada um focado em uma área crítica do desenvolvimento de backend:

| Sub-Pacote      | Descrição Principal                                                                  | Uso Típico                                                                                                             |
| :-------------- | :----------------------------------------------------------------------------------- | :--------------------------------------------------------------------------------------------------------------------- |
| **`envloader`** | **Carregamento de Configuração** de variáveis de ambiente usando **tags de struct**. | Inicialização de structs de configuração com tags `env` e `envDefault`.                                                |
| **`dyndb`**     | **Abstração Genérica de Persistência** para DynamoDB.                                | CRUD, Batch, Query fluente (`Store[T]`, `QueryBuilder`).                                                               |
| **`api`**       | **Orquestração Concorrente de APIs** e autenticação.                                 | Execução paralela de microserviços com dependências (`APIPipeline`), obtenção de tokens de segurança (`TokenService`). |

---

## Guia de início rápido (composição)

O poder da biblioteca reside na composição dos seus utilitários.

### 1\. Carregamento de configuração

Use o `envloader` para inicializar as configurações da aplicação e do `dyndb`.

```go
package main

import (
    "log"
    "[github.com/raywall/dynamodb-quick-service/envloader](https://github.com/raywall/dynamodb-quick-service/envloader)"
)

type Config struct {
    // Configuração do DynamoDB
    DynamoTableName string `env:"DYNAMODB_TABLE_NAME" envDefault:"MyDefaultTable"`

    // Configuração da API
    APIServiceHost string `env:"API_SERVICE_HOST"`
    TimeoutSeconds int    `env:"REQUEST_TIMEOUT" envDefault:"5"`
}

func main() {
    cfg := &Config{}
    envloader.MustLoad(cfg) // Panics se falhar
    log.Printf("Tabela: %s, Host: %s", cfg.DynamoTableName, cfg.APIServiceHost)
}
```

### 2\. Persistência tipada (dyndb)

Crie um Store fortemente tipado e execute consultas fluentes.

```go
package main

import (
    "context"
    "log"
    "[github.com/raywall/dynamodb-quick-service/dyndb](https://github.com/raywall/dynamodb-quick-service/dyndb)"
    // Importações do AWS SDK...
)

type Order struct {
    OrderID string `dynamodbav:"order_id"`
    Status  string `dynamodbav:"status"`
}

func GetActiveOrders(store dyndb.Store[Order]) {
    results, _, err := store.Query().
        Index("GSI_STATUS").
        KeyEqual("status", "ACTIVE").
        Limit(50).
        Exec(context.Background())

    if err != nil {
        log.Fatalf("Erro na consulta: %v", err)
    }
    log.Printf("Total de %d pedidos ativos encontrados.", len(results))
}
```

### 3\. Orquestração de microserviços (api)

Use o `APIPipeline` para orquestrar chamadas dependentes em paralelo.

```go
package main

import (
    "context"
    "log"
    "[github.com/raywall/dynamodb-quick-service/api](https://github.com/raywall/dynamodb-quick-service/api)"
)

func RunUserPipeline() {
    // API A não tem dependências
    userConfig := api.APIConfig{Name: "User", Host: "...", HttpMethod: "GET"}

    // API B depende da API A
    profileConfig := api.APIConfig{
        Name: "Profile",
        Dependencies: []string{"User"},
        Host: "...",
        HttpMethod: "POST",
    }

    pipeline := api.NewAPIPipeline([]api.APIConfig{userConfig, profileConfig})

    results, err := pipeline.Execute(context.Background(), nil)
    if err != nil {
        log.Fatalf("Pipeline falhou: %v", err)
    }

    log.Printf("Resultado do Perfil: %v", results["Profile"])
}
```
