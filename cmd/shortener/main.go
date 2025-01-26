package main

import (
	"fmt"
	"net/http"

	"github.com/atinyakov/go-url-shortener/internal/app/server"
	"github.com/atinyakov/go-url-shortener/internal/app/services"
	"github.com/atinyakov/go-url-shortener/internal/config"
	"github.com/atinyakov/go-url-shortener/internal/logger"
	"github.com/atinyakov/go-url-shortener/internal/storage"
)

func main() {

	options := config.Init()

	hostname := options.A
	resultHostname := options.B
	filePath := options.F

	log := logger.New()
	logErr := log.Init("Info")
	if logErr != nil {
		panic(logErr)
	}

	newFs := storage.FileStorage{}
	fs, fsError := newFs.Create(filePath)
	if fsError != nil {
		panic(fsError)
	}

	urlsData, fsReadError := fs.Read()
	if fsReadError != nil {
		log.Info("No file found")
	}

	var ltos, stol map[string]string
	ltos = make(map[string]string)
	stol = make(map[string]string)

	for _, record := range urlsData {
		original := record["original_url"]
		short := record["short_url"]
		ltos[original] = short
		stol[short] = original
	}

	resolver := services.NewURLResolver(8, ltos, stol)
	r := server.Init(resolver, resultHostname, log, true, fs)

	log.Info(fmt.Sprintf("Server is running on: %s", hostname))
	err := http.ListenAndServe(hostname, r)
	if err != nil {
		panic(err)
	}
}
