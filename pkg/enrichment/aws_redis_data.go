package enrichment

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Cache simples para evitar recriar clientes Redis a cada request (Opcional, mas recomendado)
var redisClients = make(map[string]*redis.Client)

func getRedisClient(addr, password string) *redis.Client {
	key := addr + password
	if client, ok := redisClients[key]; ok {
		return client
	}
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})
	redisClients[key] = client
	return client
}

// ProcessRedis executa comandos GET ou HGETALL.
func ProcessRedis(ctx context.Context, addr, password, command, key string) (interface{}, error) {
	client := getRedisClient(addr, password)

	switch command {
	case "get":
		val, err := client.Get(ctx, key).Result()
		if err == redis.Nil {
			return nil, nil // Key not found
		} else if err != nil {
			return nil, err
		}
		// Tenta fazer parse se for JSON, senão devolve string
		var data interface{}
		if err := json.Unmarshal([]byte(val), &data); err == nil {
			return data, nil
		}
		return val, nil

	case "hgetall":
		val, err := client.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		// Converte map[string]string para map[string]interface{}
		result := make(map[string]interface{})
		for k, v := range val {
			result[k] = v
		}
		return result, nil

	default:
		return nil, fmt.Errorf("comando redis não suportado: %s", command)
	}
}
