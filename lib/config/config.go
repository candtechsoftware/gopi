package config

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
	NoGit           bool
	TestPerf        bool
	TestLoadUser    bool
	TestLoadData    bool

	// User load test config
	StartUsers   int
	MaxUsers     int
	StepUsers    int
	StepDuration int

	// Data load test config
	InitialDataSize    int
	MaxDataSize        int
	DataSizeMultiplier float64
	DataStepCount      int
}

func ParseFlags() (*Config, error) {
	config := &Config{}

	flag.StringVar(&config.FilePath, "file", "", "JSON file containing endpoints")
	flag.StringVar(&config.FilePath, "f", "", "JSON file containing endpoints (shorthand)")
	flag.IntVar(&config.ThreadCount, "thread-count", 1, "Number of threads to use")
	flag.IntVar(&config.ThreadCount, "tc", 1, "Number of threads to use (shorthand)")
	flag.IntVar(&config.ConnectionCount, "connection-count", 1, "Number of connections to use")
	flag.IntVar(&config.ConnectionCount, "cc", 1, "Number of connections to use (shorthand)")
	flag.IntVar(&config.RequestCount, "request-count", 1, "Number of requests per endpoint")
	flag.IntVar(&config.RequestCount, "rc", 1, "Number of requests per endpoint (shorthand)")
	flag.BoolVar(&config.NoGit, "no-git", false, "Use timestamp-based hashes instead of git commits")

	flag.BoolVar(&config.TestPerf, "test-perf", false, "Run performance test")
	flag.BoolVar(&config.TestLoadUser, "test-load-user", false, "Run user load test")
	flag.BoolVar(&config.TestLoadData, "test-load-data", false, "Run data load test")

	// User load test flags
	flag.IntVar(&config.StartUsers, "start-users", 2, "Initial number of concurrent users")
	flag.IntVar(&config.MaxUsers, "max-users", 50, "Maximum number of concurrent users")
	flag.IntVar(&config.StepUsers, "step-users", 5, "Number of users to add per step")
	flag.IntVar(&config.StepDuration, "step-duration", 60, "Duration of each step in seconds")

	// Data load test flags
	flag.IntVar(&config.InitialDataSize, "initial-data", 1000, "Initial data size")
	flag.IntVar(&config.MaxDataSize, "max-data", 100000, "Maximum data size")
	flag.Float64Var(&config.DataSizeMultiplier, "data-multiplier", 5.0, "Data size multiplier per step")
	flag.IntVar(&config.DataStepCount, "data-steps", 4, "Number of data load steps")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: api-perf-tester [options] --test-mode

Required Test Mode (choose one):
  --test-perf           Run standard performance test
  --test-load-user      Run user connection load test
  --test-load-data      Run data volume load test

Note: For CI/CD, run test modes sequentially in separate steps.
See examples/workflows/performance.yml for reference.

Options:
  -f, --file <path>            JSON file containing endpoints
  -tc, --thread-count <num>    Number of threads to use (default: 1)
  -cc, --connection-count <num> Number of connections to use (default: 1)
  -rc, --request-count <num>    Number of requests per endpoint (default: 1)
  --no-git                     Use timestamp-based hashes instead of git commits

User Load Test Options:
  --start-users <num>          Initial number of concurrent users (default: 2)
  --max-users <num>            Maximum number of concurrent users (default: 50)
  --step-users <num>           Users to add per step (default: 5)
  --step-duration <seconds>    Duration of each step (default: 60)

Data Load Test Options:
  --initial-data <num>         Initial data size (default: 1000)
  --max-data <num>            Maximum data size (default: 100000)
  --data-multiplier <float>    Data size multiplier per step (default: 5.0)
  --data-steps <num>          Number of data load steps (default: 4)

Examples:
  api-perf-tester -f endpoints.json --test-perf
  api-perf-tester -f endpoints.json --test-load-user --start-users 5 --max-users 100
  api-perf-tester -f endpoints.json --test-load-data --initial-data 5000 --data-steps 6
`)
	}

	flag.Parse()

	if config.FilePath == "" {
		return nil, fmt.Errorf("--file or -f flag is required")
	}

	if _, err := os.Stat(config.FilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file %s does not exist", config.FilePath)
	}

	if !config.TestPerf && !config.TestLoadUser && !config.TestLoadData {
		return nil, fmt.Errorf("one test mode flag is required (--test-perf, --test-load-user, or --test-load-data)")
	}

	// Ensure only one test mode is selected
	count := 0
	if config.TestPerf {
		count++
	}
	if config.TestLoadUser {
		count++
	}
	if config.TestLoadData {
		count++
	}
	if count > 1 {
		return nil, fmt.Errorf("only one test mode can be selected at a time")
	}

	return config, nil
}
