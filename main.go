package main

import (
	"log"
	"net/http"
	"os"
	"rate_limiter/middleware"

	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
)

func main() {
	// Carregar variáveis de ambiente
	if err := godotenv.Load(); err != nil {
		log.Println("Arquivo .env não encontrado, usando variáveis de ambiente padrão.")
	}

	// Conexão com Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_HOST"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})
	defer redisClient.Close()

	// Middleware de Rate Limiter
	limiter := middleware.NewRateLimiter(redisClient)

	// Servidor HTTP
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Requisição bem-sucedida!"))
	})

	log.Println("Servidor iniciado na porta 8090.")
	log.Fatal(http.ListenAndServe(":8090", limiter.Middleware(http.DefaultServeMux)))
}
