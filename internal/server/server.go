package server

import (
	"caching-proxy/internal/proxy"
	"log"
	"net/http"
)

type Server struct {
	proxy  *proxy.Proxy
	server *http.Server
}

func New(proxy *proxy.Proxy, port string) *Server {

	router := http.NewServeMux()
	router.Handle("/", proxy)
	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}
	return &Server{
		server: server,
		proxy:  proxy,
	}
}
func (s *Server) Start() {
	log.Printf("Сервер запущен на %s", s.server.Addr)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Ошибка: %v", err)
	}
}
