// Package config provides functionality for managing configuration options
// for the application using command-line flags and environment variables.
package config

import (
	"flag"
	"os"
	"strconv"
)

// Options holds the configuration values for the application.
type Options struct {
	// Port defines the server's listening address (ip:port).
	Port string

	// ResultHostname is the base URL used for result links.
	ResultHostname string

	// FilePath is the path to the storage file for persistent data.
	FilePath string

	// DatabaseDSN holds the database connection string for the application.
	DatabaseDSN string

	// EnablePprof indicates whether to enable pprof for performance profiling.
	EnablePprof bool

	// EnableHTTPS indicates whether to enable https.
	EnableHTTPS bool
}

// options holds the current configuration values.
var options = &Options{}

// init initializes command-line flags and sets default values.
func init() {
	flag.StringVar(&options.Port, "a", "localhost:8080", "run on ip:port server")
	flag.StringVar(&options.ResultHostname, "b", "http://localhost:8080", "result base url")
	flag.StringVar(&options.FilePath, "f", "", "path to storage file")
	flag.StringVar(&options.DatabaseDSN, "d", "", "db address")
	flag.BoolVar(&options.EnablePprof, "p", false, "enable pprof")
	flag.BoolVar(&options.EnableHTTPS, "s", false, "enable https")
}

// Parse parses the command-line flags and environment variables to set
// configuration values. It returns a pointer to the Options struct containing
// the parsed configuration values.
func Parse() *Options {
	flag.Parse()

	// Override flags with environment variables if set
	if serverAddress := os.Getenv("SERVER_ADDRESS"); serverAddress != "" {
		options.Port = serverAddress
	}

	if baseURL := os.Getenv("BASE_URL"); baseURL != "" {
		options.ResultHostname = baseURL
	}

	if storagePath := os.Getenv("FILE_STORAGE_PATH"); storagePath != "" {
		options.FilePath = storagePath
	}

	if enableHTTPS := os.Getenv("ENABLE_HTTPS"); enableHTTPS != "" {
		httpMode, err := strconv.ParseBool(enableHTTPS)
		if err != nil {
			options.EnableHTTPS = false
		}

		options.EnableHTTPS = httpMode
	}

	return options
}
