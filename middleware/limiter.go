package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

type RateLimiter struct {
	redisClient *redis.Client
	ctx         context.Context
}

func NewRateLimiter(redisClient *redis.Client) *RateLimiter {
	return &RateLimiter{
		redisClient: redisClient,
		ctx:         context.Background(),
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("API_KEY")
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)

		// Determinar limites
		limit, ttl := rl.GetLimits(ip, token)
		key := rl.GetKey(ip, token)

		// Incrementar contador
		count, err := rl.redisClient.Incr(rl.ctx, key).Result()
		if err != nil {
			http.Error(w, "Erro interno", http.StatusInternalServerError)
			return
		}

		// Configurar TTL no Redis
		if count == 1 {
			rl.redisClient.Expire(rl.ctx, key, ttl)
		}

		// Bloquear requisições excedentes
		if count > int64(limit) {
			http.Error(w, "429 - you have reached the maximum number of requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) GetLimits(ip, token string) (int, time.Duration) {
	// Configurações padrão
	defaultLimit, _ := strconv.Atoi(os.Getenv("DEFAULT_LIMIT"))
	defaultTTL, _ := strconv.Atoi(os.Getenv("DEFAULT_TTL"))

	// Verificar configurações do token
	if token != "" {
		limit, err := rl.redisClient.HGet(rl.ctx, "token:"+token, "limit").Int()
		ttl, errTTL := rl.redisClient.HGet(rl.ctx, "token:"+token, "ttl").Int()
		if err == nil && errTTL == nil {
			return limit, time.Duration(ttl) * time.Second
		}
	}

	// Retornar limites padrão
	return defaultLimit, time.Duration(defaultTTL) * time.Second
}

func (rl *RateLimiter) GetKey(ip, token string) string {
	if token != "" {
		return fmt.Sprintf("rate-limiter:token:%s", token)
	}
	return fmt.Sprintf("rate-limiter:ip:%s", ip)
}
