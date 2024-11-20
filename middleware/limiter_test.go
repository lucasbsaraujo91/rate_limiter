package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"rate_limiter/middleware"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiterMiddleware(t *testing.T) {
	// Configura variáveis de ambiente
	os.Setenv("DEFAULT_LIMIT", "5")
	os.Setenv("DEFAULT_TTL", "60")

	// Mock Redis
	mockRedis := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Limpa o Redis antes do teste
	mockRedis.FlushAll(mockRedis.Context())

	// Cria o RateLimiter
	limiter := middleware.NewRateLimiter(mockRedis)

	// Rota simulada
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Success"))
	})
	handler := limiter.Middleware(testHandler)

	t.Run("Request under limit", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
		assert.Equal(t, "Success", w.Body.String())
	})

	t.Run("Request exceeding limit", func(t *testing.T) {
		ip := "192.168.1.2"
		for i := 0; i < 6; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = ip + ":12345"
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if i < 5 {
				assert.Equal(t, http.StatusOK, w.Result().StatusCode)
			} else {
				assert.Equal(t, http.StatusTooManyRequests, w.Result().StatusCode)
				assert.Equal(t, "429 - you have reached the maximum number of requests\n", w.Body.String())
			}
		}
	})

	t.Run("Custom token limits", func(t *testing.T) {
		// Configura limite para um token específico
		token := "custom_token"
		mockRedis.HSet(mockRedis.Context(), "token:"+token, "limit", 2)
		mockRedis.HSet(mockRedis.Context(), "token:"+token, "ttl", 30)

		for i := 0; i < 3; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("API_KEY", token)
			req.RemoteAddr = "192.168.1.3:12345"
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if i < 2 {
				assert.Equal(t, http.StatusOK, w.Result().StatusCode)
			} else {
				assert.Equal(t, http.StatusTooManyRequests, w.Result().StatusCode)
			}
		}
	})
}

func TestGetKey(t *testing.T) {
	limiter := middleware.NewRateLimiter(nil)

	t.Run("Key with token", func(t *testing.T) {
		token := "abc123"
		expected := "rate-limiter:token:abc123"
		actual := limiter.GetKey("", token)
		assert.Equal(t, expected, actual)
	})

	t.Run("Key with IP", func(t *testing.T) {
		ip := "192.168.1.1"
		expected := "rate-limiter:ip:192.168.1.1"
		actual := limiter.GetKey(ip, "")
		assert.Equal(t, expected, actual)
	})
}

func TestGetLimits(t *testing.T) {
	os.Setenv("DEFAULT_LIMIT", "10")
	os.Setenv("DEFAULT_TTL", "120")
	mockRedis := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer mockRedis.Close()
	limiter := middleware.NewRateLimiter(mockRedis)

	t.Run("Default limits", func(t *testing.T) {
		limit, ttl := limiter.GetLimits("", "")
		assert.Equal(t, 10, limit)
		assert.Equal(t, 120*time.Second, ttl)
	})

	t.Run("Token-specific limits", func(t *testing.T) {
		token := "test_token"
		mockRedis.HSet(mockRedis.Context(), "token:"+token, "limit", 5)
		mockRedis.HSet(mockRedis.Context(), "token:"+token, "ttl", 60)

		limit, ttl := limiter.GetLimits("", token)
		assert.Equal(t, 5, limit)
		assert.Equal(t, 60*time.Second, ttl)
	})
}
