package server

import (
	"cacheProxy/internal/config"
	"cacheProxy/internal/handlers"
	"cacheProxy/internal/middleware"
	"log"
	"net/http"

	Cache "cacheProxy/internal/cache"

	"github.com/gin-gonic/gin"
)

type Server struct {
	cfg    *config.Config
	cache  Cache.Cache
	router *gin.Engine
}

func NewServer(cfg *config.Config, cache Cache.Cache) *Server {
	// Создаем роутер Gin внутри конструктора
	router := gin.Default()

	return &Server{
		cfg:    cfg,
		cache:  cache,
		router: router,
	}
}
func (s *Server) setupRoutes() {
	// 1. Создаем прокси-хендлер, передавая URL реального сервера из нашего конфига
	proxyHandler := handlers.NewProxyHandler(s.cfg.Server.UpstreamURL)

	// 2. Подключаем наше Middleware глобально и передаем TTL кэша из конфига
	s.router.Use(middleware.CacheMiddleware(s.cache, s.cfg.Server.CacheTTL))

	// 3. Сервисные эндпоинты (благодаря исключению внутри middleware, они не проксируются)
	s.router.GET("/stats", func(c *gin.Context) {
		c.JSON(http.StatusOK, s.cache.Stats())
	})

	s.router.POST("/clear", func(c *gin.Context) {
		if err := s.cache.Clear(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "cache cleared"})
	})

	// 4. Все остальные запросы отправляем на проксирование
	s.router.NoRoute(proxyHandler.ProxyRequest)
}

// Run настраивает роуты и запускает HTTP-сервер на порту из конфигурации
func (s *Server) Run() error {
	// Инициализируем маршруты прямо перед запуском
	s.setupRoutes()

	log.Printf("Server is starting on port %s", s.cfg.Server.Port)

	// Запускаем сервер на указанном в конфиге порту (например, ":8080")
	return s.router.Run(s.cfg.Server.Port)
}
