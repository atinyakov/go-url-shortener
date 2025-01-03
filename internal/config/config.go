package config

import "flag"

type Options struct {
	A string
	B string
}

func Init() *Options {
	options := &Options{}

	flag.StringVar(&options.A, "a", "localhost:8080", "run on ip:port server")
	flag.StringVar(&options.B, "b", "http://localhost:8080", "result base url")

	return options
}
