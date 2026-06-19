package storage

import (
	"cacheProxy/internal/cache"
	"cacheProxy/internal/config"
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
	ctx    context.Context
	mu     *sync.RWMutex
	stats  cache.Stats
}

func NewRedisCache(ctx context.Context, cfg *config.Config) (*RedisCache, error) {
	client, err := newConnection(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &RedisCache{
		client: client,
		ctx:    context.Background(),
		mu:     &sync.RWMutex{},
		stats: cache.Stats{
			LastCleared: time.Now(),
		},
	}, nil
}
func (r *RedisCache) Set(key string, item *cache.CacheItem) error {
	data, err := item.MarshalBinary()
	if err != nil {
		return err
	}
	if err = r.client.Set(r.ctx, key, data, item.TTL).Err(); err != nil {
		return err
	}
	return nil
}
func (r *RedisCache) Get(key string) (*cache.CacheItem, bool) {
	// 1. Читаем строку из Redis
	val, err := r.client.Get(r.ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		r.mu.Lock()
		r.stats.Misses++
		r.mu.Unlock()
		return nil, false
	} else if err != nil {
		r.mu.Lock()
		r.stats.Misses++ // Ошибки сети тоже логично считать за промах
		r.mu.Unlock()
		return nil, false
	}

	// 2. Десериализуем обратно в структуру
	var item cache.CacheItem
	if err := json.Unmarshal([]byte(val), &item); err != nil {
		r.mu.Lock()
		r.stats.Misses++
		r.mu.Unlock()
		return nil, false
	}
	r.mu.Lock()
	r.stats.Hits++
	r.mu.Unlock()
	return &item, true
}
func (r *RedisCache) Delete(key string) error {
	if err := r.client.Del(r.ctx, key).Err(); err != nil {
		return err
	}
	return nil
}
func (r *RedisCache) Clear() error {
	if err := r.client.FlushDB(r.ctx).Err(); err != nil {
		return err
	}
	// Сбрасываем счетчики
	r.mu.Lock()
	r.stats.Hits = 0
	r.stats.Misses = 0
	r.stats.Evictions = 0
	r.stats.LastCleared = time.Now()
	r.mu.Unlock()

	return nil
}
func (r *RedisCache) Stats() *cache.Stats {
	// Делаем быстрый запрос в Redis, чтобы узнать реальное количество ключей прямо сейчас
	dbSize, _ := r.client.DBSize(r.ctx).Result()

	r.mu.RLock() // Блокируем на чтение
	defer r.mu.RUnlock()

	// Возвращаем копию, чтобы избежать race condition при чтении полей снаружи
	return &cache.Stats{
		Hits:        r.stats.Hits,
		Misses:      r.stats.Misses,
		Evictions:   r.stats.Evictions,
		LastCleared: r.stats.LastCleared,
		Size:        int(dbSize), // Актуальный размер базы из Redis
	}
}
func newConnection(ctx context.Context, cfg *config.Config) (*redis.Client, error) {
	db := redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.Addr,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		Username:     cfg.Redis.User,
		MaxRetries:   cfg.Redis.MaxRetries,
		DialTimeout:  cfg.Redis.DialTimeout,
		ReadTimeout:  cfg.Redis.Timeout,
		WriteTimeout: cfg.Redis.Timeout,
	})
	if err := db.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return db, nil
}
