package config

import (
	"flag"
	"os"
)

type Options struct {
	Port           string
	ResultHostname string
	FilePath       string
	DatabaseDSN    string
}

var options = &Options{}

func init() {
	flag.StringVar(&options.Port, "a", "localhost:8080", "run on ip:port server")
	flag.StringVar(&options.ResultHostname, "b", "http://localhost:8080", "result base url")
	flag.StringVar(&options.FilePath, "f", "", "path to storage file")
	flag.StringVar(&options.DatabaseDSN, "d", "", "db address")
}

func Parse() *Options {
	flag.Parse()

	if serverAddress := os.Getenv("SERVER_ADDRESS"); serverAddress != "" {
		options.Port = serverAddress
	}

	if baseURL := os.Getenv("BASE_URL"); baseURL != "" {
		options.ResultHostname = baseURL
	}

	if storagePath := os.Getenv("FILE_STORAGE_PATH"); storagePath != "" {
		options.FilePath = storagePath
	}

	return options
}
