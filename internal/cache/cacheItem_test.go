package cache

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestCacheItem_IsExpired(t *testing.T) {
	tests := []struct {
		name    string
		created time.Time
		ttl     time.Duration
		want    bool
	}{
		{"свежая запись", time.Now(), time.Minute, false},
		{"срок истёк", time.Now().Add(-2 * time.Minute), time.Minute, true},
		{"нулевой TTL", time.Now().Add(-time.Second), 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &CacheItem{CreatedAt: tt.created, TTL: tt.ttl}
			if got := item.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCacheItem_MarshalRoundTrip(t *testing.T) {
	orig := &CacheItem{
		Body:      []byte(`{"id":1}`),
		Headers:   http.Header{"Content-Type": {"application/json"}},
		Status:    http.StatusOK,
		CreatedAt: time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC),
		TTL:       time.Minute,
	}

	data, err := orig.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v", err)
	}

	var got CacheItem
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if string(got.Body) != string(orig.Body) ||
		got.Status != orig.Status ||
		got.TTL != orig.TTL ||
		!got.CreatedAt.Equal(orig.CreatedAt) ||
		got.Headers.Get("Content-Type") != "application/json" {
		t.Errorf("round trip = %+v, want %+v", got, orig)
	}
}
