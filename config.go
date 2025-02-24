package main

import (
	"flag"
	"fmt"
	"os"
)

type Config struct {
	FilePath        string
	ThreadCount     int
	ConnectionCount int
	RequestCount    int
}

func ParseFlags() (*Config, error) {
	config := &Config{}

	flag.StringVar(&config.FilePath, "file", "", "JSON file containing endpoints")
	flag.StringVar(&config.FilePath, "f", "", "JSON file containing endpoints (shorthand)")
	flag.IntVar(&config.ThreadCount, "thread-count", 1, "Number of threads to use")
	flag.IntVar(&config.ConnectionCount, "connection-count", 1, "Number of connections to use")
	flag.IntVar(&config.RequestCount, "request-count", 1, "Number of requests per endpoint")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: api-perf-tester [options]\n\nOptions:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if err := validate(config); err != nil {
		return nil, err
	}

	return config, nil
}

func validate(config *Config) error {
	if config.FilePath == "" {
		return fmt.Errorf("--file flag is required")
	}

	if _, err := os.Stat(config.FilePath); os.IsNotExist(err) {
		return fmt.Errorf("file %s does not exist", config.FilePath)
	}

	return nil
}
