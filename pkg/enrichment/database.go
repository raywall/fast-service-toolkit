package enrichment

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq" // Driver Postgres (Importe outros no main.go se necessário)
)

// ProcessSQL executa uma query e retorna uma lista de mapas.
// driver: "postgres", "mysql", etc.
// dsn: Connection string.
func ProcessSQL(ctx context.Context, driver, dsn, query string, args []interface{}) ([]map[string]interface{}, error) {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir conexão SQL: %w", err)
	}
	defer db.Close()

	// Timeout de segurança para banco de dados
	ctxDb, cancel := context.WithTimeout(ctx, 5*time.Second) // Poderia ser configurável
	defer cancel()

	rows, err := db.QueryContext(ctxDb, query, args...)
	if err != nil {
		return nil, fmt.Errorf("erro na query SQL: %w", err)
	}
	defer rows.Close()

	// Mapeamento dinâmico de colunas para mapa
	columns, _ := rows.Columns()
	count := len(columns)
	values := make([]interface{}, count)
	valuePtrs := make([]interface{}, count)

	var finalResult []map[string]interface{}

	for rows.Next() {
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		entry := make(map[string]interface{})
		for i, col := range columns {
			var v interface{}
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				v = string(b)
			} else {
				v = val
			}
			entry[col] = v
		}
		finalResult = append(finalResult, entry)
	}

	return finalResult, nil
}
