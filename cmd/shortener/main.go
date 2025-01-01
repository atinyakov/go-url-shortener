package main

import (
	"net/http"

	"github.com/atinyakov/go-url-shortener/internal/app/server"
	"github.com/atinyakov/go-url-shortener/internal/app/services"
)

func main() {
	resolver := services.NewURLResolver(8)
	r := server.Init(resolver)

	err := http.ListenAndServe(`:8080`, r)
	if err != nil {
		panic(err)
	}
}
