# Etapa de build
FROM golang:1.23 AS builder
WORKDIR /app

# Copiar módulos e instalar dependências
COPY go.mod go.sum ./
RUN go mod download

# Copiar código-fonte
COPY . .

# Compilar aplicação com CGO desabilitado e usando `musl` para compatibilidade com Alpine
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o rate_limiter .

# Etapa final
FROM alpine:latest
WORKDIR /root/

# Instalar dependências necessárias
RUN apk --no-cache add ca-certificates

# Copiar binário compilado
COPY --from=builder /app/rate_limiter .

# Garantir permissão de execução ao binário
RUN chmod +x rate_limiter

# Porta exposta e comando padrão
EXPOSE 8080
CMD ["./rate_limiter"]

