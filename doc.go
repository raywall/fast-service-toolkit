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
//
// Package fast_service_lab (Fast Service Toolkit) é um framework High-Code/Low-Code
// declarativo para construção rápida, segura e padronizada de serviços Backend
// (REST e GraphQL) e BFFs (Backends for Frontends).
//
// Visão Geral:
// O Toolkit inverte o paradigma tradicional de desenvolvimento: em vez de escrever
// código imperativo (handlers, services, repositories) para cada endpoint, o
// desenvolvedor define o *comportamento* do serviço via configuração (YAML/JSON).
// O framework se encarrega da orquestração, validação, conectividade e observabilidade.
//
// O núcleo do framework baseia-se em uma arquitetura de Pipeline de Execução
// alimentada por um motor de regras CEL (Common Expression Language), permitindo
// lógica de negócio complexa, transformações de dados e validações sem recompilação.
//
// Modos de Operação:
//
//  1. REST Service Mode:
//     Focado em endpoints HTTP tradicionais. Define um pipeline linear de execução:
//     Input -> Validation -> Enrichment (Data Fetching) -> Processing (Business Logic) -> Output.
//     Ideal para APIs de agregação, transformadores de dados e proxies inteligentes.
//
//  2. GraphQL Mesh Mode:
//     Atua como um Gateway de federação ou orquestração. Permite definir schemas
//     GraphQL onde os resolvers de cada campo são mapeados declarativamente para
//     fontes de dados externas (REST, DynamoDB, Fixed), resolvendo automaticamente
//     o grafo de dependências e concorrência.
//
//  3. Interceptor Mode (Smart Proxy):
//     Atua como um Middleware Arquitetural (Ambassador Pattern) em frente a outros
//     microsserviços. Ele recebe a requisição, enriquece o payload com dados de
//     múltiplas fontes, aplica validações de negócio e encaminha a requisição
//     "turbinada" para o serviço de destino, desacoplando a lógica de carga de dados.
//
// Componentes Principais (pkg):
//
//   - pkg/engine: O cérebro do framework. Gerencia o ciclo de vida da requisição,
//     carrega configurações e coordena os middlewares e steps.
//
//   - pkg/rules: Wrapper sobre o Google CEL. Oferece um ambiente seguro para execução
//     de expressões lógicas (`input.age > 18`) e transformações (`vars.total * 0.1`)
//     dentro da configuração YAML.
//
//   - pkg/enrichment: Camada de acesso a dados (Data Sources). Fornece conectores
//     otimizados e instrumentados para REST, GraphQL, AWS DynamoDB, S3, Secrets
//     Manager e Parameter Store.
//
//   - pkg/auth: Gerenciamento de autenticação OAuth2 (Client Credentials) transparente.
//     Gerencia automaticamente a obtenção, cache e renovação de tokens para chamadas
//     externas.
//
//   - pkg/transport: Abstração de entrada (Entrypoints). Permite que o mesmo "Engine"
//     rode transparentemente em servidores HTTP locais (Gin/NetHttp), AWS Lambda
//     (API Gateway/SQS) ou Containers ECS/EKS.
//
// Exemplo de Definição Declarativa (Interceptor):
//
//	service:
//	  name: "loan-enricher"
//	  runtime: "lambda"
//
//	middlewares:
//	  - type: "enrichment"
//	    config:
//	      sources:
//	        - name: "redis_data"
//	          type: "aws_redis"
//	          params: { key: "'USER:' + input.cpf" }
//
//	steps:
//	  output:
//	    body:
//	      original_req: "input"
//	      enrichment: "detection.redis_data"
//	    target:
//	      url: "'http://pricing-service.internal/calc'"
//	      method: "POST"
//
// Filosofia:
// "Configuration as Code, Logic as Expression". O Toolkit visa eliminar o código
// repetitivo de infraestrutura, permitindo que times foquem puramente na regra
// de negócio e na integração de sistemas.
package fast_service_toolkit
