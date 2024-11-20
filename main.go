package main

import (
	"log"
	"net/http"
	"os"
	"rate_limiter/middleware"
	redisstorage "rate_limiter/storage"

	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Arquivo .env não encontrado, usando variáveis de ambiente padrão.")
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_HOST"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})
	defer redisClient.Close()

	redisStorage := redisstorage.NewRedisStorage(redisClient) // Passando o ponteiro corretamente
	limiter := middleware.NewRateLimiter(redisStorage)        // Passando o ponteiro

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Requisição bem-sucedida!"))
	})

	log.Println("Servidor iniciado na porta 8080.")
	log.Fatal(http.ListenAndServe(":8080", limiter.Middleware(http.DefaultServeMux)))
}
