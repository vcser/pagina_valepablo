FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copiar go.mod y go.sum primero (cache de dependencias)
COPY go.mod ./
RUN go mod download

# Copiar el resto del código
COPY . .

# Compilar binario
RUN go build -o server .

# Imagen final
FROM alpine:3.19

WORKDIR /app

# Crear volumen para archivos estáticos
VOLUME ["/app"]

COPY --from=builder /app/server /app/server

EXPOSE 8081

CMD ["/app/server"]
