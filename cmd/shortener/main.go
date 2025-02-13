package main

import (
	"errors"
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
	dbName := options.DatabaseDSN

	var s storage.StorageI

	log := logger.New()
	err := log.Init("Info")
	if err != nil {
		panic(err)
	}

	if dbName != "" {
		db := repository.InitDB(dbName)
		defer db.Close()
		log.Info("using db")
		s = repository.CreateURLRepository(db)
	} else if filePath != "" {
		log.Info("using file")

		s, err = storage.NewFileStorage(filePath)
		if err != nil {
			panic(err)
		}
	} else {
		log.Info("using in memory storage")

		s, err = storage.CreateMemoryStorage()
		if err != nil {
			panic(err)
		}
	}

	if s == nil {
		panic(errors.New("NO STORAGE"))

	}

	resolver, err := services.NewURLResolver(8, s)
	if err != nil {
		panic(err)
	}
	r := server.Init(resolver, resultHostname, log, true, s)

	log.Info(fmt.Sprintf("Server is running on: %s", hostname))
	err = http.ListenAndServe(hostname, r)
	if err != nil {
		panic(err)
	}
}
