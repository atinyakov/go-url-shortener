package config

import (
	"flag"
	"os"
)

type Options struct {
	A string
	B string
}

func Init() *Options {
	options := &Options{}

	flag.StringVar(&options.A, "a", "localhost:8080", "run on ip:port server")
	flag.StringVar(&options.B, "b", "http://localhost:8080", "result base url")

	flag.Parse()

	if serverAddress := os.Getenv("SERVER_ADDRESS"); serverAddress != "" {
		options.A = serverAddress
	}

	if baseURL := os.Getenv("BASE_URL"); baseURL != "" {
		options.B = baseURL
	}

	return options
}
