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
// Package emulator fornece um servidor HTTP mock inteligente, configurável via JSON,
// projetado para acelerar o desenvolvimento local, testes de integração e validação
// de cenários de borda sem depender de serviços reais.
//
// Visão Geral:
// O `emulator` permite simular APIs REST completas sem a necessidade de escrever
// código boilerplate (handlers, routers) para cada endpoint. Ele suporta múltiplas
// portas simultâneas, rotas estáticas e dinâmicas, e uma lógica de filtragem de
// dados em memória.
//
// Diferente de mocks estáticos simples (que retornam sempre o mesmo JSON), o
// emulator possui um motor de regras capaz de cruzar parâmetros de URL (Path) e
// Query String com um dataset pré-definido, retornando respostas contextuais
// (ex: 200 com o objeto específico encontrado, ou 404 se não houver match).
//
// Funcionalidades Principais:
//   - Multi-Server: Capacidade de subir múltiplos servidores em portas diferentes
//     simultaneamente (ex: simular Microsserviço A na 8080 e B na 8081).
//   - Roteamento Dinâmico: Suporte a variáveis no path (ex: `/users/{id}`).
//   - Filtragem de Dados: Mapeamento automático de `query_params` e `path_params`
//     para campos do dataset JSON configurado.
//   - Respostas Condicionais: Configuração distinta para "Match" (dado encontrado)
//     e "No Match" (404/Erro).
//   - Simulação de Latência/Erro: Pode ser configurado para simular cenários de falha.
//
// Estrutura de Configuração (JSON):
// O emulador é controlado por um array de objetos `ServerConfig`.
//
//	[
//	  {
//	    "port": 8080,
//	    "routes": [ ... ]
//	  }
//	]
//
// Exemplos de Uso:
//
// Exemplo de Configuração (config.json):
// Este exemplo define uma rota que busca usuários por ID (Path) ou Tipo (Query).
//
//	[
//	  {
//	    "port": 8080,
//	    "routes": [
//	      {
//	        "path": "/users/{id}",
//	        "method": "GET",
//	        "path_params": [{ "name": "id", "maps_to": "user_id" }],
//	        "data": [
//	          { "user_id": 1, "name": "Alice", "role": "admin" },
//	          { "user_id": 2, "name": "Bob", "role": "user" }
//	        ],
//	        "response_on_match": { "status": 200 },
//	        "response_on_no_match": {
//	          "status": 404,
//	          "body": { "error": "User not found" }
//	        }
//	      }
//	    ]
//	  }
//	]
//
// Exemplo de Inicialização Programática (Go):
//
//	package main
//
//	import (
//	    "log"
//	    "sync"
//	    "github.com/raywall/fast-service-toolkit/tools/emulator/config"
//	)
//
//	func main() {
//	    var cfg config.Config
//	    if err := cfg.LoadFromFile("config.json"); err != nil {
//	        log.Fatal(err)
//	    }
//
//	    var wg sync.WaitGroup
//	    for _, server := range []config.ServerConfig(cfg) {
//	        wg.Add(1)
//	        go func(s config.ServerConfig) {
//	            defer wg.Done()
//	            s.Start() // Inicia o servidor HTTP
//	        }(server)
//	    }
//	    wg.Wait()
//	}
package emulator
