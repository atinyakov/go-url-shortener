package main

import (
	"fmt"
	"net/http"

	"github.com/atinyakov/go-url-shortener/internal/app/server"
	"github.com/atinyakov/go-url-shortener/internal/app/service"
	"github.com/atinyakov/go-url-shortener/internal/config"
	"github.com/atinyakov/go-url-shortener/internal/logger"
	"github.com/atinyakov/go-url-shortener/internal/repository"
	"github.com/atinyakov/go-url-shortener/internal/storage"
	"go.uber.org/zap"
)

func main() {

	options := config.Parse()

	hostname := options.Port
	resultHostname := options.ResultHostname
	filePath := options.FilePath
	dbName := options.DatabaseDSN

	var s service.Storage

	log := logger.New()
	err := log.Init("Info")
	zapLogger := log.Log
	if err != nil {
		panic(err)
	}

	if dbName != "" {
		zapLogger.Info(fmt.Sprintf("using db %s", dbName))
		db := repository.InitDB(dbName)
		defer db.Close()
		s = repository.CreateURLRepository(db, zapLogger)
		zapLogger.Info("Database connected and table ready.")
		s = repository.CreateURLRepository(db, zapLogger)
		zapLogger.Info("Database connected and table ready.")
	} else if filePath != "" {
		zapLogger.Info("using file", zap.String("filePath", filePath))

		s, err = storage.NewFileStorage(filePath, zapLogger)
		if err != nil {
			panic(err)
		}
	} else {
		zapLogger.Info("using in memory storage")

		s, err = storage.CreateMemoryStorage()
		if err != nil {
			panic(err)
		}
	}

	resolver, err := service.NewURLResolver(8, s)
	if err != nil {
		panic(err)
	}
	URLService := service.NewURL(s, resolver, resultHostname)
	r := server.Init(resultHostname, zapLogger, true, URLService)

	zapLogger.Info("Server is running", zap.String("hostname", hostname))
	err = http.ListenAndServe(hostname, r)
	if err != nil {
		panic(err)
	}
}
