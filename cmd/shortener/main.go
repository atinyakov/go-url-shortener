package main

import (
	"fmt"
	"net/http"

	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"

	"github.com/atinyakov/go-url-shortener/internal/app/server"
	"github.com/atinyakov/go-url-shortener/internal/app/service"
	"github.com/atinyakov/go-url-shortener/internal/config"
	"github.com/atinyakov/go-url-shortener/internal/logger"
	"github.com/atinyakov/go-url-shortener/internal/repository"
	"github.com/atinyakov/go-url-shortener/internal/storage"

	_ "net/http/pprof"
)

var buildVersion string
var buildDate string
var buildCommit string

func main() {
	options := config.Parse()
	hostname := options.Port
	resultHostname := options.ResultHostname
	filePath := options.FilePath
	dbName := options.DatabaseDSN
	useTLS := options.EnableHTTPS

	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)

	var s service.Storage

	log := logger.New()
	defer func() {
		_ = log.Log.Sync()
	}()

	err := log.Init("Info")
	zapLogger := log.Log
	if err != nil {
		panic(err)
	}

	if options.EnablePprof {
		go func() {
			zapLogger.Info("Starting pprof server", zap.String("addr", "localhost:6060"))
			if err := http.ListenAndServe("localhost:6060", nil); err != nil {
				zapLogger.Error("pprof server error", zap.Error(err))
			}
		}()
	}

	if dbName != "" {
		zapLogger.Info("using db", zap.String("dbName", dbName))
		db := repository.InitDB(dbName, zapLogger)
		defer db.Close()
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

	URLService := service.NewURL(s, resolver, zapLogger, resultHostname)
	r := server.Init(resultHostname, zapLogger, true, URLService)

	if useTLS {
		manager := &autocert.Manager{
			// директория для хранения сертификатов
			Cache: autocert.DirCache("cache-dir"),
			// функция, принимающая Terms of Service издателя сертификатов
			Prompt: autocert.AcceptTOS,
			// перечень доменов, для которых будут поддерживаться сертификаты
			HostPolicy: autocert.HostWhitelist("mysite.ru", "www.mysite.ru"),
		}
		// конструируем сервер с поддержкой TLS
		server := &http.Server{
			Addr:    ":443",
			Handler: r,
			// для TLS-конфигурации используем менеджер сертификатов
			TLSConfig: manager.TLSConfig(),
		}
		zapLogger.Info("Server is running with TLS", zap.String("hostname", hostname))
		server.ListenAndServeTLS("", "")
	} else {
		zapLogger.Info("Server is running", zap.String("hostname", hostname))
		err = http.ListenAndServe(hostname, r)

		if err != nil {
			panic(err)
		}
	}
}
