package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"rate_limiter/middleware"
	redisstorage "rate_limiter/storage"

	// Corrigido para o nome correto do pacote
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
)

func setupRedis() *redis.Client {
	mockRedis := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	mockRedis.FlushAll(mockRedis.Context()) // Limpa o Redis antes dos testes
	return mockRedis
}

func TestRateLimiterMiddleware(t *testing.T) {
	os.Setenv("DEFAULT_LIMIT", "5")
	os.Setenv("DEFAULT_TTL", "60")

	redisClient := setupRedis()
	defer redisClient.Close()

	storage := redisstorage.NewRedisStorage(redisClient) // Usando o pacote correto
	limiter := middleware.NewRateLimiter(storage)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Success"))
	})
	handler := limiter.Middleware(testHandler)

	t.Run("Request under limit", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode, "Deveria permitir requisição dentro do limite")
		assert.Equal(t, "Success", w.Body.String(), "Resposta inesperada")
	})

	t.Run("Request exceeding limit", func(t *testing.T) {
		ip := "192.168.1.2"
		for i := 0; i < 6; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = ip + ":12345"
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if i < 5 {
				assert.Equal(t, http.StatusOK, w.Result().StatusCode, "Deveria permitir até 5 requisições")
			} else {
				assert.Equal(t, http.StatusTooManyRequests, w.Result().StatusCode, "Deveria bloquear após exceder o limite")
				assert.Contains(t, w.Body.String(), "429 - you have reached the maximum number of requests", "Mensagem de erro incorreta")
			}
		}
	})

	t.Run("Custom token limits", func(t *testing.T) {
		token := "custom_token"
		redisClient.HSet(redisClient.Context(), "token:"+token, "limit", 2)
		redisClient.HSet(redisClient.Context(), "token:"+token, "ttl", 30)

		for i := 0; i < 3; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("API_KEY", token)
			req.RemoteAddr = "192.168.1.3:12345"
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if i < 2 {
				assert.Equal(t, http.StatusOK, w.Result().StatusCode, "Deveria permitir até 2 requisições")
			} else {
				assert.Equal(t, http.StatusTooManyRequests, w.Result().StatusCode, "Deveria bloquear após exceder o limite de 2 requisições")
			}
		}
	})
}

func TestGetKey(t *testing.T) {
	redisClient := setupRedis()
	defer redisClient.Close()
	storage := redisstorage.NewRedisStorage(redisClient) // Usando o pacote correto
	limiter := middleware.NewRateLimiter(storage)

	t.Run("Key with token", func(t *testing.T) {
		token := "abc123"
		expected := "rate-limiter:token:abc123"
		actual := limiter.GetKey("", token)
		assert.Equal(t, expected, actual, "Chave gerada para token está incorreta")
	})

	t.Run("Key with IP", func(t *testing.T) {
		ip := "192.168.1.1"
		expected := "rate-limiter:ip:192.168.1.1"
		actual := limiter.GetKey(ip, "")
		assert.Equal(t, expected, actual, "Chave gerada para IP está incorreta")
	})
}

func TestGetLimits(t *testing.T) {
	os.Setenv("DEFAULT_LIMIT", "10")
	os.Setenv("DEFAULT_TTL", "120")

	redisClient := setupRedis()
	defer redisClient.Close()

	storage := redisstorage.NewRedisStorage(redisClient) // Usando o pacote correto
	limiter := middleware.NewRateLimiter(storage)

	t.Run("Default limits", func(t *testing.T) {
		limit, ttl := limiter.GetLimits("", "")
		assert.Equal(t, 10, limit, "Limite padrão está incorreto")
		assert.Equal(t, 120*time.Second, ttl, "TTL padrão está incorreto")
	})

	t.Run("Token-specific limits", func(t *testing.T) {
		token := "test_token"
		redisClient.HSet(redisClient.Context(), "token:"+token, "limit", 5)
		redisClient.HSet(redisClient.Context(), "token:"+token, "ttl", 60)

		limit, ttl := limiter.GetLimits("", token)
		assert.Equal(t, 5, limit, "Limite para token específico está incorreto")
		assert.Equal(t, 60*time.Second, ttl, "TTL para token específico está incorreto")
	})
}
