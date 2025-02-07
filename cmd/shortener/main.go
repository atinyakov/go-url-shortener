package main

import (
	"fmt"
	"net/http"

	"github.com/atinyakov/go-url-shortener/internal/app/server"
	"github.com/atinyakov/go-url-shortener/internal/app/services"
	"github.com/atinyakov/go-url-shortener/internal/config"
	"github.com/atinyakov/go-url-shortener/internal/logger"
	"github.com/atinyakov/go-url-shortener/internal/repository"
	"github.com/atinyakov/go-url-shortener/internal/storage"
)

func main() {

	options := config.Init()

	hostname := options.Port
	resultHostname := options.ResultHostname
	filePath := options.FilePath
	dbAddress := options.DatabaseDSN

	db := repository.InitDb(dbAddress)
	defer db.Close()

	log := logger.New()
	logErr := log.Init("Info")
	if logErr != nil {
		panic(logErr)
	}

	fs, fsError := storage.NewFileStorage(filePath)
	if fsError != nil {
		panic(fsError)
	}

	resolver, err := services.NewURLResolver(8, fs)
	if err != nil {
		panic(err)
	}
	r := server.Init(resolver, resultHostname, log, true, fs, db)

	log.Info(fmt.Sprintf("Server is running on: %s", hostname))
	err = http.ListenAndServe(hostname, r)
	if err != nil {
		panic(err)
	}
}
