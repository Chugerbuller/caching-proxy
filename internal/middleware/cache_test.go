package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	Cache "cacheProxy/internal/cache"

	"github.com/gin-gonic/gin"
)

// fakeCache — кэш в памяти вместо Redis, чтобы тестировать логику middleware
// без внешних сервисов.
type fakeCache struct {
	mu    sync.Mutex
	items map[string]*Cache.CacheItem
	sets  int
}

func newFakeCache() *fakeCache {
	return &fakeCache{items: make(map[string]*Cache.CacheItem)}
}

func (f *fakeCache) Set(key string, item *Cache.CacheItem) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.items[key] = item
	f.sets++
	return nil
}

func (f *fakeCache) Get(key string) (*Cache.CacheItem, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	item, ok := f.items[key]
	return item, ok
}

func (f *fakeCache) Delete(key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.items, key)
	return nil
}

func (f *fakeCache) Clear() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.items = make(map[string]*Cache.CacheItem)
	return nil
}

func (f *fakeCache) Stats() *Cache.Stats {
	f.mu.Lock()
	defer f.mu.Unlock()
	return &Cache.Stats{Size: len(f.items)}
}

// upstream считает, сколько раз запрос дошёл до «реального сервера».
type upstream struct {
	mu    sync.Mutex
	calls int
}

func (u *upstream) count() int {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.calls
}

func (u *upstream) hit() {
	u.mu.Lock()
	u.calls++
	u.mu.Unlock()
}

func newRouter(c Cache.Cache, up *upstream) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CacheMiddleware(c, time.Minute))

	r.GET("/posts/:id", func(ctx *gin.Context) {
		up.hit()
		ctx.Header("X-Upstream", "yes")
		ctx.JSON(http.StatusOK, gin.H{"id": ctx.Param("id"), "q": ctx.Query("q")})
	})
	r.GET("/broken", func(ctx *gin.Context) {
		up.hit()
		ctx.String(http.StatusInternalServerError, "boom")
	})
	r.POST("/posts", func(ctx *gin.Context) {
		up.hit()
		ctx.Status(http.StatusCreated)
	})
	r.GET("/stats", func(ctx *gin.Context) {
		up.hit()
		ctx.JSON(http.StatusOK, c.Stats())
	})
	return r
}

func get(r http.Handler, method, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec
}

func TestCacheMiddleware_MissThenHit(t *testing.T) {
	c := newFakeCache()
	up := &upstream{}
	r := newRouter(c, up)

	first := get(r, http.MethodGet, "/posts/1")
	if first.Code != http.StatusOK {
		t.Fatalf("first status = %d, want 200", first.Code)
	}
	if got := first.Header().Get("X-Cache"); got != "MISS" {
		t.Errorf("first X-Cache = %q, want MISS", got)
	}

	second := get(r, http.MethodGet, "/posts/1")
	if got := second.Header().Get("X-Cache"); got != "HIT" {
		t.Errorf("second X-Cache = %q, want HIT", got)
	}
	if up.count() != 1 {
		t.Errorf("upstream calls = %d, want 1 (второй запрос должен прийти из кэша)", up.count())
	}
	if second.Code != http.StatusOK {
		t.Errorf("second status = %d, want 200", second.Code)
	}
	if second.Body.String() != first.Body.String() {
		t.Errorf("cached body = %q, want %q", second.Body, first.Body)
	}
	if ct := second.Header().Get("Content-Type"); ct != first.Header().Get("Content-Type") {
		t.Errorf("cached Content-Type = %q, want %q", ct, first.Header().Get("Content-Type"))
	}
	if second.Header().Get("X-Upstream") != "yes" {
		t.Error("cached response lost upstream headers")
	}
}

func TestCacheMiddleware_StoresItemWithTTL(t *testing.T) {
	c := newFakeCache()
	r := newRouter(c, &upstream{})

	get(r, http.MethodGet, "/posts/7")

	item, ok := c.Get("/posts/7")
	if !ok {
		t.Fatal("response was not stored in cache")
	}
	if item.Status != http.StatusOK || item.TTL != time.Minute || len(item.Body) == 0 {
		t.Errorf("stored item = status %d, ttl %v, body %d bytes", item.Status, item.TTL, len(item.Body))
	}
}

func TestCacheMiddleware_QueryIsPartOfKey(t *testing.T) {
	c := newFakeCache()
	up := &upstream{}
	r := newRouter(c, up)

	get(r, http.MethodGet, "/posts/1?q=a")
	rec := get(r, http.MethodGet, "/posts/1?q=b")

	if got := rec.Header().Get("X-Cache"); got != "MISS" {
		t.Errorf("X-Cache = %q, want MISS for different query", got)
	}
	if up.count() != 2 {
		t.Errorf("upstream calls = %d, want 2", up.count())
	}
}

func TestCacheMiddleware_DoesNotCacheErrors(t *testing.T) {
	c := newFakeCache()
	up := &upstream{}
	r := newRouter(c, up)

	get(r, http.MethodGet, "/broken")
	rec := get(r, http.MethodGet, "/broken")

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if up.count() != 2 {
		t.Errorf("upstream calls = %d, want 2 (ошибки не кэшируются)", up.count())
	}
	if c.sets != 0 {
		t.Errorf("cache sets = %d, want 0", c.sets)
	}
}

func TestCacheMiddleware_SkipsNonGET(t *testing.T) {
	c := newFakeCache()
	up := &upstream{}
	r := newRouter(c, up)

	get(r, http.MethodPost, "/posts")
	rec := get(r, http.MethodPost, "/posts")

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", rec.Code)
	}
	if rec.Header().Get("X-Cache") != "" {
		t.Errorf("X-Cache = %q, want empty for POST", rec.Header().Get("X-Cache"))
	}
	if up.count() != 2 || c.sets != 0 {
		t.Errorf("upstream calls = %d, cache sets = %d; want 2 and 0", up.count(), c.sets)
	}
}

func TestCacheMiddleware_SkipsServicePaths(t *testing.T) {
	c := newFakeCache()
	up := &upstream{}
	r := newRouter(c, up)

	get(r, http.MethodGet, "/stats")
	get(r, http.MethodGet, "/stats")

	if up.count() != 2 {
		t.Errorf("/stats upstream calls = %d, want 2 (служебный путь не кэшируется)", up.count())
	}
	if _, ok := c.Get("/stats"); ok {
		t.Error("/stats must not be stored in cache")
	}
}
