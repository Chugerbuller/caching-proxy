package middleware

import (
	"bytes"
	"log"
	"net/http"
	"time"

	Cache "cacheProxy/internal/cache"

	"github.com/gin-gonic/gin"
)

type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w responseBodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func CacheMiddleware(cache Cache.Cache, ttl time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Лог 1: Запрос только что прилетел в прокси
		log.Printf("[Middleware] ---> Входящий запрос: %s %s", c.Request.Method, c.Request.URL.Path)

		if c.Request.URL.Path == "/stats" || c.Request.URL.Path == "/clear" {
			log.Printf("[Middleware] Пропуск кэша для сервисного пути: %s", c.Request.URL.Path)
			c.Next()
			return
		}
		if c.Request.Method != http.MethodGet {
			log.Printf("[Middleware] Пропуск кэша: метод %s не GET", c.Request.Method)
			c.Next()
			return
		}

		cacheKey := c.Request.URL.RequestURI()
		log.Printf("[Middleware] Сгенерирован ключ кэша: %s", cacheKey)

		// Проверяем Redis
		if item, found := cache.Get(cacheKey); found {
			// Лог 2: Ура, нашли в кэше!
			log.Printf("[Middleware] [HIT] Ключ найден в Redis. Отдаем закэшированный ответ.")

			for k, vv := range item.Headers {
				for _, v := range vv {
					c.Writer.Header().Add(k, v)
				}
			}
			c.Writer.Header().Set("X-Cache", "HIT")
			c.Data(item.Status, c.Writer.Header().Get("Content-Type"), item.Body)
			c.Abort()
			return
		}

		// Лог 3: В Redis пусто, идем дальше
		log.Printf("[Middleware] [MISS] Ключа нет в Redis. Ставим X-Cache=MISS и готовимся к проксированию.")
		c.Writer.Header().Set("X-Cache", "MISS")

		blw := &responseBodyWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		// Лог 4: Передаем управление хендлеру прокси
		log.Printf("[Middleware] Замораживаем middleware, уходим в c.Next()...")
		c.Next()

		// Лог 5: Хендлер закончил работу, мы вернулись
		log.Printf("[Middleware] Вернулись из c.Next(). Статус ответа от реального сервера: %d", c.Writer.Status())

		if c.Writer.Status() == http.StatusOK {
			log.Printf("[Middleware] Статус 200 OK. Записываем %d байт в Redis для ключа %s", blw.body.Len(), cacheKey)
			cacheItem := &Cache.CacheItem{
				Body:      blw.body.Bytes(),
				Headers:   c.Writer.Header().Clone(),
				Status:    c.Writer.Status(),
				CreatedAt: time.Now(),
				TTL:       ttl,
			}
			_ = cache.Set(cacheKey, cacheItem)
		} else {
			log.Printf("[Middleware] Статус не 200 (%d), пропускаем сохранение в Redis", c.Writer.Status())
		}

		log.Printf("[Middleware] <--- Обработка запроса %s завершена", cacheKey)
	}
}
