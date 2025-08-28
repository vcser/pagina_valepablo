# Etapa de construcción
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copiar dependencias
COPY go.mod ./
RUN go mod download

# Copiar todo el código, templates y static
COPY . .

# Compilar binario
RUN go build -o server .

# Imagen final mínima
FROM alpine:3.19

WORKDIR /app

# ⚡ NO montar volumen en /app para no borrar templates/static
# VOLUMES pueden ir solo si quieres persistir subcarpetas de datos generados
# VOLUME ["/app/data"]

# Copiar binario
COPY --from=builder /app/server /app/server

# Copiar templates y static
COPY --from=builder /app/templates /app/templates
COPY --from=builder /app/static /app/static

EXPOSE 8081

CMD ["/app/server"]