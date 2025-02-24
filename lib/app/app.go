package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"percipio.com/gopi/lib/config"
	"percipio.com/gopi/lib/history"
	"percipio.com/gopi/lib/logger"
	"percipio.com/gopi/lib/runner"
	"percipio.com/gopi/lib/stats"
	"percipio.com/gopi/lib/viz"
)

type App struct {
	runner       *runner.Runner
	config       *config.Config
	historyStore *history.Store
}

type EndpointConfig struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

type TestConfig []EndpointConfig

func New() (*App, error) {
	logger.Info("Initializing application...")
	cfg, err := config.ParseFlags()
	if err != nil {
		return nil, err
	}

	testConfig, err := loadTestConfig(cfg.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load test config: %w", err)
	}

	benchRunner := runner.NewRunner(cfg.ThreadCount, cfg.RequestCount)

	for _, endpoint := range testConfig {
		task := runner.Task{
			URL:     endpoint.URL,
			Method:  endpoint.Method,
			Headers: endpoint.Headers,
		}
		if endpoint.Body != "" {
			task.Body = []byte(endpoint.Body)
		}
		benchRunner.AddTask(task)
	}

	logger.Info("Loaded %d endpoints from config file", len(testConfig))

	historyStore, err := history.NewStore("", 10.0, !cfg.NoGit)
	if err != nil {
		logger.Warn("Failed to initialize history store: %v. Continuing without history tracking.", err)
		historyStore = nil
	}

	return &App{
		runner:       benchRunner,
		config:       cfg,
		historyStore: historyStore,
	}, nil
}

func loadTestConfig(filepath string) (TestConfig, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config TestConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if len(config) == 0 {
		return nil, fmt.Errorf("no endpoints defined in config file")
	}

	return config, nil
}

func (a *App) Run() {
	switch {
	case a.config.TestPerf:
		logger.Info("Running performance test...")
		a.runStandardTest()
	case a.config.TestLoadUser:
		logger.Info("Running user load test...")
		a.runUserLoadTest()
	case a.config.TestLoadData:
		logger.Info("Running data load test...")
		a.runDataLoadTest()
	}
}

// Move existing Run() logic to this method
func (a *App) runStandardTest() {
	logger.Info("Starting performance test...")
	results := a.runner.Run()
	statistics := stats.Calculate(results)

	var testHistory *history.TestHistory
	if a.historyStore != nil {
		var err error
		testHistory, err = a.historyStore.SaveResults(statistics)
		if err != nil {
			logger.Error("Failed to save test history: %v", err)
		}
	}

	// Print current test results
	logger.Info("Performance test completed")
	for endpoint, stats := range statistics.EndpointStats {
		fmt.Printf("\nEndpoint: %s\n", endpoint)
		fmt.Printf("  Average Latency: %.2fms\n", float64(stats.AverageDuration.Milliseconds()))
		fmt.Printf("  P50 Latency: %.2fms\n", float64(stats.P50Latency.Milliseconds()))
		fmt.Printf("  P95 Latency: %.2fms\n", float64(stats.P95Latency.Milliseconds()))
		fmt.Printf("  P99 Latency: %.2fms\n", float64(stats.P99Latency.Milliseconds()))
		fmt.Printf("  Requests/sec: %.2f\n", stats.RequestsPerSecond)
		fmt.Printf("  Success Rate: %.2f%%\n", successRate(stats))
	}

	// Only show historical comparisons if we have a history store and test history
	if a.historyStore != nil && testHistory != nil {
		if testHistory.Degradation {
			logger.Warn("Performance degradation detected!")
			fmt.Printf("\nPerformance Comparison (Baseline: %s)\n", testHistory.BaselineID)
			for endpoint, comparison := range testHistory.Endpoints {
				if comparison.Degradation {
					fmt.Printf("\nEndpoint: %s\n", endpoint)
					fmt.Printf("  Latency Increase: %.2f%%\n", comparison.Changes.LatencyIncrease)
					fmt.Printf("  Error Rate Increase: %.2f%%\n", comparison.Changes.ErrorRateIncrease)
					fmt.Printf("  Throughput Decrease: %.2f%%\n", comparison.Changes.ThroughputDecrease)
					fmt.Printf("  Success Rate Decrease: %.2f%%\n", comparison.Changes.SuccessRateDecrease)
				}
			}
		}

		// Try to generate graphs
		summary, err := a.historyStore.GetSummary()
		if err != nil {
			logger.Error("Failed to load performance summary: %v", err)
		} else {
			reportPath, err := viz.GenerateGraph(summary, "performance-reports")
			if err != nil {
				logger.Error("Failed to generate performance graphs: %v", err)
			} else {
				absPath, _ := filepath.Abs(reportPath)
				logger.Info("Performance graphs generated in performance-reports directory")
				fmt.Printf("\nView results at: file://%s\n", absPath)
			}
		}
	}
}

func (a *App) runUserLoadTest() {
	logger.Info("Starting user load test...")

	config := runner.UserLoadConfig{
		StartUsers:      a.config.StartUsers,
		MaxUsers:        a.config.MaxUsers,
		StepUsers:       a.config.StepUsers,
		DurationPerStep: time.Duration(a.config.StepDuration) * time.Second,
	}

	logger.Info("Load test configuration:")
	logger.Info("- Starting with %d users", config.StartUsers)
	logger.Info("- Maximum users: %d", config.MaxUsers)
	logger.Info("- Step size: %d users", config.StepUsers)
	logger.Info("- Step duration: %v", config.DurationPerStep)
	logger.Info("- Total steps: %d", (config.MaxUsers-config.StartUsers)/config.StepUsers+1)

	results := a.runner.RunUserLoadTest(config)
	loadStats := stats.CalculateLoadTest(results)

	if a.historyStore != nil {
		if _, err := a.historyStore.SaveLoadTestResults(loadStats, history.TestTypeLoadUser); err != nil {
			logger.Error("Failed to save load test history: %v", err)
		}
	}

	fmt.Printf("\nUser Load Test Summary\n")
	fmt.Printf("====================\n")
	fmt.Printf("Total Duration: %v\n", loadStats.TestDuration)
	fmt.Printf("Total Requests: %d\n", loadStats.TotalRequests)
	fmt.Printf("Overall Average Latency: %v\n\n", loadStats.AverageLatency)

	fmt.Printf("Step-by-Step Results:\n")
	fmt.Printf("-------------------\n")
	for _, step := range loadStats.Steps {
		fmt.Printf("Concurrent Users: %d\n", step.UserCount)
		fmt.Printf("  Average Latency: %v\n", step.AverageLatency)
		fmt.Printf("  Requests/sec: %.2f\n", step.RequestsPerSecond)
		fmt.Printf("  Success Rate: %.2f%%\n", step.SuccessRate)
		fmt.Printf("  Error Rate: %.2f%%\n\n", step.ErrorRate)
	}
}

func (a *App) runDataLoadTest() {
	logger.Info("Starting data load test...")

	config := runner.DataLoadConfig{
		InitialDataSize:    a.config.InitialDataSize,
		MaxDataSize:        a.config.MaxDataSize,
		DataSizeMultiplier: a.config.DataSizeMultiplier,
		StepsCount:         a.config.DataStepCount,
	}

	logger.Info("Data load test configuration:")
	logger.Info("- Initial size: %d", config.InitialDataSize)
	logger.Info("- Maximum size: %d", config.MaxDataSize)
	logger.Info("- Size multiplier: %.1fx", config.DataSizeMultiplier)
	logger.Info("- Number of steps: %d", config.StepsCount)

	results := a.runner.RunDataLoadTest(config)
	loadStats := stats.CalculateLoadTest(results)

	if a.historyStore != nil {
		if _, err := a.historyStore.SaveLoadTestResults(loadStats, history.TestTypeLoadData); err != nil {
			logger.Error("Failed to save load test history: %v", err)
		}
	}

	fmt.Printf("\nData Load Test Summary\n")
	fmt.Printf("=====================\n")
	fmt.Printf("Total Duration: %v\n", loadStats.TestDuration)
	fmt.Printf("Total Requests: %d\n", loadStats.TotalRequests)
	fmt.Printf("Overall Average Latency: %v\n\n", loadStats.AverageLatency)

	fmt.Printf("Step-by-Step Results:\n")
	fmt.Printf("-------------------\n")
	for _, step := range loadStats.Steps {
		fmt.Printf("Data Size: %d records\n", step.DataSize)
		fmt.Printf("  Average Latency: %v\n", step.AverageLatency)
		fmt.Printf("  Requests/sec: %.2f\n", step.RequestsPerSecond)
		fmt.Printf("  Success Rate: %.2f%%\n", step.SuccessRate)
		fmt.Printf("  Error Rate: %.2f%%\n\n", step.ErrorRate)
	}
}

func successRate(stats *stats.EndpointStatistics) float64 {
	if stats.TotalRequests == 0 {
		return 0
	}
	return float64(stats.SuccessRequests) / float64(stats.TotalRequests) * 100
}
