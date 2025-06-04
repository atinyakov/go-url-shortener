package main

import (
	"cmp"
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

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
	trustedSubnet := options.TrustedSubnet

	fmt.Printf("Build version: %s\n", cmp.Or(buildVersion, "N/A"))
	fmt.Printf("Build date: %s\n", cmp.Or(buildDate, "N/A"))
	fmt.Printf("Build commit: %s\n", cmp.Or(buildCommit, "N/A"))

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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	URLService, shutdown := service.NewURL(ctx, s, resolver, zapLogger, resultHostname)
	defer shutdown()

	router := server.Init(resultHostname, trustedSubnet, zapLogger, true, URLService)

	var srv *http.Server

	if useTLS {
		manager := &autocert.Manager{
			Cache:      autocert.DirCache("cache-dir"),
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist("mysite.ru", "www.mysite.ru"),
		}
		srv = &http.Server{
			Addr:      ":443",
			Handler:   router,
			TLSConfig: manager.TLSConfig(),
		}
		go func() {
			zapLogger.Info("Server is running with TLS", zap.String("hostname", hostname))
			if err := srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				zapLogger.Fatal("Server error", zap.Error(err))
			}
		}()
	} else {
		srv = &http.Server{
			Addr:    hostname,
			Handler: router,
		}
		go func() {
			zapLogger.Info("Server is running", zap.String("hostname", hostname))
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				zapLogger.Fatal("Server error", zap.Error(err))
			}
		}()
	}

	// Ожидаем сигнал завершения
	<-ctx.Done()
	zapLogger.Info("Shutdown signal received")

	// Завершаем сервер с таймаутом
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		zapLogger.Error("Server shutdown error", zap.Error(err))
	} else {
		zapLogger.Info("Server shutdown gracefully")
	}
}
