# Этап 1: Сборка бинарника
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Сначала копируем зависимости для кэширования слоев Docker
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь исходный код проекта
COPY . .

# Собираем приложение из папки cmd/, где лежит main.go
RUN go build -o /app/my-go-app ./cmd/main.go

# Этап 2: Финальный минимальный образ
FROM alpine:latest
WORKDIR /root/

# Копируем скомпилированный бинарник
COPY --from=builder /app/my-go-app .

# Копируем файл конфигурации, чтобы приложение могло его прочитать
COPY config.yaml .

# Команда для запуска приложения
CMD ["./my-go-app"]