package cache

import (
	"encoding/json"
	"net/http"
	"time"
)

type CacheItem struct { //todo name correctly
	Body      []byte        `json:"body"`
	Headers   http.Header   `json:"headers"`
	Status    int           `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
	TTL       time.Duration `json:"ttl"`
}

func (c *CacheItem) IsExpired() bool {
	return time.Since(c.CreatedAt) > c.TTL
}
func (c *CacheItem) MarshalBinary() ([]byte, error) {
	res, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	return res, nil
}
