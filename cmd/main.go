package main

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"cacheProxy/internal/config"
	"cacheProxy/internal/server"
	"cacheProxy/internal/storage"
)

func main() {
	// 1. Загружаем конфигурацию (предположим, у тебя есть функция Load)
	cfg := config.Init()

	// 2. Инициализируем Redis, используя параметры из cfg
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Проверяем соединение с Redis перед стартом
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}

	// 3. Инициализируем хранилище кэша
	cacheStorage, err := storage.NewRedisCache(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to initialize cache storage: %v", err)
	}

	// 4. Создаем наш сервер, передавая ему конфиг и хранилище кэша
	srv := server.NewServer(cfg, cacheStorage)
	log.Printf("Cache Proxy Server is starting on port %s, forwarding to %s", cfg.Server.Port, cfg.Server.UpstreamURL)
	// 5. Запускаем сервер
	if err := srv.Run(); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}
