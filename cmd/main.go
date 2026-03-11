package main

import (
	"caching-proxy/internal/proxy"
	"caching-proxy/internal/server"
	"flag"
	"fmt"
	"os"
)

func main() {
	PORT := flag.String("port", "0", "Define PORT on which the caching proxy server will run")
	ORIGIN := flag.String("origin", "", "Define the URL of the server to which the requests will be forwarded")
	CLEAR_CACHE := flag.Bool("clear-cache", false, "Clear the Cache")
	flag.Parse()

	proxy := proxy.New(*ORIGIN)
	if *CLEAR_CACHE {
		fmt.Println("Clearing Cache....")
		proxy.Cache.Clear()
		os.Exit(0)
	}
	server := server.New(proxy, *PORT)
	server.Start()
}
