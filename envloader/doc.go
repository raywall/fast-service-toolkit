// Package envloader fornece um utilitário simples para carregar variáveis de
// ambiente diretamente para campos de uma struct Go, incluindo suporte
// para tags de ambiente (`env`) e valores padrão (`envDefault`).
//
// Visão Geral:
// O `envloader` simplifica a gestão de configurações em aplicações Go.
// Ele utiliza reflection para inspecionar a struct de configuração e mapear
// automaticamente variáveis de ambiente para os campos tipados. Suporta tipos
// básicos como string, int, uint, bool e float, além de structs aninhadas
// (incluindo ponteiros para structs).
//
// Funcionalidades Principais:
// - Mapeamento por Tag: Usa a tag `env:"VAR_NAME"` para encontrar a variável.
// - Valores Padrão: Usa a tag `envDefault:"value"` se a variável não estiver definida.
// - Suporte a Aninhamento: Processa structs aninhadas e ponteiros para structs.
// - Tratamento de Erros Tipados: Retorna erros específicos para configurações inválidas ou conversões de tipo.
//
// Exemplos de Uso:
//
// Exemplo Básico:
// Demonstra como carregar uma configuração simples.
//
//   // Assumindo que a variável de ambiente DB_HOST está definida como "localhost"
//   type Config struct {
//       DBHost string `env:"DB_HOST"`
//       DBPort int    `env:"DB_PORT" envDefault:"5432"`
//   }
//   
//   var cfg Config
//   if err := envloader.Load(&cfg); err != nil {
//       log.Fatal(err)
//   }
//
//   fmt.Printf("Host: %s, Port: %d\n", cfg.DBHost, cfg.DBPort) // Output: Host: localhost, Port: 5432
//
// Exemplo com Struct Aninhada:
//
//   type ServerConfig struct {
//       Host string `env:"SERVER_HOST"`
//   }
//   type AppConfig struct {
//       Server ServerConfig
//   }
//   
//   // Assumindo SERVER_HOST="0.0.0.0" está definido
//   var appCfg AppConfig
//   if err := envloader.Load(&appCfg); err != nil {
//       log.Fatal(err)
//   }
//
// Configuração:
// O pacote requer que a função `Load` receba um ponteiro para a struct de configuração.
// As variáveis de ambiente devem estar definidas no sistema operacional antes da execução de `Load`.
package envloader