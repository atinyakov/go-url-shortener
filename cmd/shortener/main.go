package main

import (
	"net/http"

	"github.com/atinyakov/go-url-shortener/internal/app/handlers"
	"github.com/atinyakov/go-url-shortener/internal/app/services"
)

func handle(resolver *services.URLResolver) http.HandlerFunc {
	handler := handlers.NewURLHandler(resolver)
	return func(res http.ResponseWriter, req *http.Request) {

		if req.Method == http.MethodPost {
			handler.HandlePost(res, req)
		}

		if req.Method == http.MethodGet {
			handler.HandleGet(res, req)
		}

		res.WriteHeader(http.StatusBadRequest)

	}
}

func main() {
	mux := http.NewServeMux()
	resolver := services.NewURLResolver(8)

	mux.HandleFunc(`/`, handle(resolver))

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}
