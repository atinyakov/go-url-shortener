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

	options := config.Parse()

	hostname := options.Port
	resultHostname := options.ResultHostname
	filePath := options.FilePath
	dbName := options.DatabaseDSN

	var s services.Storage

	log := logger.New()
	err := log.Init("Info")
	if err != nil {
		panic(err)
	}

	if dbName != "" {
		log.Info(fmt.Sprintf("using db %s", dbName))
		db := repository.InitDB(dbName)
		defer db.Close()
		s = repository.CreateURLRepository(db, log)
		log.Info("Database connected and table ready.")
	} else if filePath != "" {
		log.Info(fmt.Sprintf("using file %s", filePath))

		s, err = storage.NewFileStorage(filePath, log)
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

	resolver, err := services.NewURLResolver(8, s)
	if err != nil {
		panic(err)
	}
	URLService := services.NewURLService(s, resolver, resultHostname)
	r := server.Init(resultHostname, log, true, URLService)

	log.Info(fmt.Sprintf("Server is running on: %s", hostname))
	err = http.ListenAndServe(hostname, r)
	if err != nil {
		panic(err)
	}
}
