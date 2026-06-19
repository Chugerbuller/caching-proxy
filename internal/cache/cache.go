package cache

//TODO прокинуть контекст
import (
	"time"
)

type Stats struct {
	Hits        int64     `json:"hits"`
	Misses      int64     `json:"misses"`
	Evictions   int64     `json:"evictions"`
	LastCleared time.Time `json:"last_cleared"`
	Size        int       `json:"size"`
}
type Cache interface {
	Set(key string, item *CacheItem) error
	Get(key string) (*CacheItem, bool)
	Delete(key string) error
	Clear() error
	Stats() *Stats
}
