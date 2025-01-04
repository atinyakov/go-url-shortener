package main

import (
	"fmt"
	"net/http"

	"github.com/atinyakov/go-url-shortener/internal/app/server"
	"github.com/atinyakov/go-url-shortener/internal/app/services"
	"github.com/atinyakov/go-url-shortener/internal/config"
)

func main() {

	options := config.Init()

	hostname := options.A
	resultHostname := options.B

	resolver := services.NewURLResolver(8)
	r := server.Init(resolver, resultHostname)

	fmt.Println("Server is running on:", hostname)
	err := http.ListenAndServe(hostname, r)
	if err != nil {
		panic(err)
	}
}
