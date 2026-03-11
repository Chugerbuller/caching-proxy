package proxy

import (
	"caching-proxy/internal/cache"
	"io"
	"log"
	"maps"
	"net/http"
	"time"
)

const (
	defaultExpiration = 5 * time.Minute
	cleanupInterval   = 3 * time.Minute
	HIT               = "HIT"
	MISS              = "MISS"
)

type Proxy struct {
	Origin string
	Cache  *cache.Cache
	Client *http.Client
}
type ProxyCacheItem struct {
	Response *http.Response
	Body     []byte
}

func New(origin string) *Proxy {
	cache := cache.New(defaultExpiration, cleanupInterval)
	return &Proxy{
		Origin: origin,
		Cache:  cache,
		Client: &http.Client{},
	}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.String() == "/clear-csh" {
		p.Cache.Clear()
		w.Write([]byte("Cash was cleared"))
		return
	}
	cacheKey := r.Method + ":" + r.URL.String()

	if item, found := p.Cache.Get(cacheKey); found {
		if value, ok := item.Value.(ProxyCacheItem); ok {
			log.Print("ok")
			RespondWithHeaders(w, value.Response, value.Body, HIT)
		}
		return
	}
	originURL := p.Origin + r.URL.String()
	log.Println(originURL)
	req, err := http.NewRequest(
		r.Method, originURL, nil,
	)
	resp, err := p.Client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	cacheItem := ProxyCacheItem{
		Response: resp,
		Body:     body,
	}
	p.Cache.Set(cacheKey, cacheItem, 5*time.Minute)
	RespondWithHeaders(w, resp, body, MISS)
}
func RespondWithHeaders(w http.ResponseWriter, response *http.Response, body []byte, cacheHeader string) {
	w.Header().Set("X-CACHE", cacheHeader)
	w.WriteHeader(response.StatusCode)
	maps.Copy(w.Header(), response.Header)
	_, err := w.Write(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
