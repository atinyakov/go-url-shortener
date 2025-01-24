package main

import (
	"fmt"
	"net/http"

	"github.com/atinyakov/go-url-shortener/internal/app/server"
	"github.com/atinyakov/go-url-shortener/internal/app/services"
	"github.com/atinyakov/go-url-shortener/internal/config"
	"github.com/atinyakov/go-url-shortener/internal/logger"
)

func main() {

	options := config.Init()

	hostname := options.A
	resultHostname := options.B

	log := logger.New()
	logErr := log.Init("Info")
	if logErr != nil {
		panic(logErr)
	}

	resolver := services.NewURLResolver(8)
	r := server.Init(resolver, resultHostname, log, true)

	fmt.Println("Server is running on:", hostname)
	err := http.ListenAndServe(hostname, r)
	if err != nil {
		panic(err)
	}
}
