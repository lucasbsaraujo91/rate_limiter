package middleware

import (
	"fmt"
	"net"
	"net/http"
	"os"
	redisstorage "rate_limiter/storage"
	"strconv"
	"time"
)

type RateLimiter struct {
	storage redisstorage.RedisStorage
}

func NewRateLimiter(storage redisstorage.RedisStorage) *RateLimiter {
	return &RateLimiter{storage: storage}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("API_KEY")
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)

		// Determinar limites
		limit, ttl := rl.GetLimits(ip, token)
		key := rl.GetKey(ip, token)

		// Incrementar contador
		count, err := rl.storage.Increment(key)
		if err != nil {
			http.Error(w, "Erro interno", http.StatusInternalServerError)
			return
		}

		// Configurar TTL no Redis
		if count == 1 {
			rl.storage.Expire(key, ttl)
		}

		// Bloquear requisições excedentes
		if count > int64(limit) {
			http.Error(w, "429 - you have reached the maximum number of requests or actions allowed", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) GetLimits(ip, token string) (int, time.Duration) {
	defaultLimit, _ := strconv.Atoi(os.Getenv("DEFAULT_LIMIT"))
	defaultTTL, _ := strconv.Atoi(os.Getenv("DEFAULT_TTL"))

	if token != "" {
		limit, ttl, err := rl.storage.GetTokenLimits(token)
		if err == nil {
			return limit, ttl
		}
	}

	return defaultLimit, time.Duration(defaultTTL) * time.Second
}

func (rl *RateLimiter) GetKey(ip, token string) string {
	if token != "" {
		return fmt.Sprintf("rate-limiter:token:%s", token)
	}
	return fmt.Sprintf("rate-limiter:ip:%s", ip)
}
