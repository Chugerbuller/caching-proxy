package handlers

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

type ProxyHandler struct {
	proxy *httputil.ReverseProxy
}

func NewProxyHandler(upstream string) *ProxyHandler {
	// 1. Парсим URL целевого сервера
	target, err := url.Parse(upstream)
	if err != nil {
		log.Fatalf("Ошибка парсинга upstream URL: %v", err)
	}

	// 2. Создаем встроенный в Go ReverseProxy
	proxy := httputil.NewSingleHostReverseProxy(target)

	// 3. Кастомизируем директор запросов (Director).
	// Director — это функция, которая модифицирует входящий запрос перед отправкой на upstream.
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		// Вызываем дефолтный директор, который настроит URL схемы (http/https)
		originalDirector(req)

		// КРИТИЧЕСКИ ВАЖНО для обхода 503 ошибок:
		// Устанавливаем Host заголовка равным хосту целевого сервера (например, "httpbin.org").
		// Без этого удаленный сервер отклонит запрос, так как в Host будет висеть "localhost:8080".
		req.Host = target.Host
	}

	return &ProxyHandler{
		proxy: proxy,
	}
}

// ProxyRequest теперь невероятно простой и надежный
func (p *ProxyHandler) ProxyRequest(c *gin.Context) {
	// Нам больше не нужно вручную копировать заголовки, читать тело и делать http.Client.Do!
	// httputil.ReverseProxy сделает всё за нас под капотом.
	// Он также автоматически запишет данные в наш кастомный responseBodyWriter,
	// благодаря чему кэш в middleware успешно сохранится!
	p.proxy.ServeHTTP(c.Writer, c.Request)
}
