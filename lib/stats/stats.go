package stats

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"percipio.com/gopi/lib/runner"
)

type EndpointStatistics struct {
	URL               string
	Method            string
	TotalRequests     int
	SuccessRequests   int
	FailedRequests    int
	TotalDuration     time.Duration
	AverageDuration   time.Duration
	MinDuration       time.Duration
	MaxDuration       time.Duration
	MedianDuration    time.Duration
	Percentile95      time.Duration
	Percentile99      time.Duration
	RequestsPerSecond float64
	StatusCodes       map[int]int
	SuccessCodes      int
	ClientErrors      int
	ServerErrors      int
	P50Latency        time.Duration
	P95Latency        time.Duration
	P99Latency        time.Duration
}

type Statistics struct {
	EndpointStats map[string]*EndpointStatistics
	TotalRequests int
	TotalDuration time.Duration
}

type LoadTestStats struct {
	Steps          []StepStatistics     `json:"steps"`
	EndpointStats  map[string]LoadStats `json:"endpointStats"`
	AverageLatency time.Duration        `json:"averageLatency"`
	MaxLatency     time.Duration        `json:"maxLatency"`
	MinLatency     time.Duration        `json:"minLatency"`
	TotalRequests  int                  `json:"totalRequests"`
	TestDuration   time.Duration        `json:"testDuration"`
}

type StepStatistics struct {
	UserCount         int           `json:"userCount,omitempty"`
	DataSize          int           `json:"dataSize,omitempty"`
	AverageLatency    time.Duration `json:"averageLatency"`
	RequestsPerSecond float64       `json:"requestsPerSecond"`
	SuccessRate       float64       `json:"successRate"`
	ErrorRate         float64       `json:"errorRate"`
}

type LoadStats struct {
	AverageLatency    time.Duration `json:"averageLatency"`
	P50Latency        time.Duration `json:"p50Latency"`
	P95Latency        time.Duration `json:"p95Latency"`
	P99Latency        time.Duration `json:"p99Latency"`
	RequestsPerSecond float64       `json:"requestsPerSecond"`
	SuccessRate       float64       `json:"successRate"`
	MaxConcurrent     int           `json:"maxConcurrent"`
	MaxDataSize       int           `json:"maxDataSize"`
}

func Calculate(results []runner.Result) *Statistics {
	stats := &Statistics{
		EndpointStats: make(map[string]*EndpointStatistics),
	}

	for _, result := range results {
		key := fmt.Sprintf("%s %s", result.Method, result.URL)
		if _, exists := stats.EndpointStats[key]; !exists {
			stats.EndpointStats[key] = &EndpointStatistics{
				URL:         result.URL,
				Method:      result.Method,
				MinDuration: time.Hour,
				StatusCodes: make(map[int]int),
			}
		}

		endpointStat := stats.EndpointStats[key]
		endpointStat.TotalRequests++
		stats.TotalRequests++

		if result.Error != nil {
			endpointStat.FailedRequests++
			continue
		}

		endpointStat.SuccessRequests++
		endpointStat.TotalDuration += result.Duration
		stats.TotalDuration += result.Duration

		if result.Duration < endpointStat.MinDuration {
			endpointStat.MinDuration = result.Duration
		}
		if result.Duration > endpointStat.MaxDuration {
			endpointStat.MaxDuration = result.Duration
		}

		if result.Error == nil {
			endpointStat.StatusCodes[result.StatusCode]++
			switch {
			case result.StatusCode >= 200 && result.StatusCode < 300:
				endpointStat.SuccessCodes++
			case result.StatusCode >= 400 && result.StatusCode < 500:
				endpointStat.ClientErrors++
			case result.StatusCode >= 500:
				endpointStat.ServerErrors++
			}
		}
	}

	for _, stat := range stats.EndpointStats {
		calculateEndpointStats(stat, results)
	}

	return stats
}

func calculateEndpointStats(stat *EndpointStatistics, results []runner.Result) {
	var durations []time.Duration
	for _, result := range results {
		if result.URL == stat.URL && result.Error == nil {
			durations = append(durations, result.Duration)
		}
	}

	if len(durations) > 0 {
		sort.Slice(durations, func(i, j int) bool {
			return durations[i] < durations[j]
		})

		stat.AverageDuration = time.Duration(stat.TotalDuration.Nanoseconds() / int64(stat.SuccessRequests))
		stat.MedianDuration = durations[len(durations)/2]
		stat.Percentile95 = durations[int(float64(len(durations))*0.95)]
		stat.Percentile99 = durations[int(float64(len(durations))*0.99)]
		stat.RequestsPerSecond = float64(stat.SuccessRequests) / stat.TotalDuration.Seconds()
		stat.calculatePercentiles(durations)
	}
}

func (s *EndpointStatistics) calculatePercentiles(durations []time.Duration) {
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	l := len(durations)
	s.P50Latency = durations[l*50/100]
	s.P95Latency = durations[l*95/100]
	s.P99Latency = durations[l*99/100]
}

func (s *Statistics) String() string {
	var sb strings.Builder
	sb.WriteString("Performance Test Summary\n")
	sb.WriteString("=======================\n")
	sb.WriteString(fmt.Sprintf("Total Requests: %d\n", s.TotalRequests))
	sb.WriteString(fmt.Sprintf("Total Duration: %v\n\n", s.TotalDuration))

	for _, stat := range s.EndpointStats {
		sb.WriteString(fmt.Sprintf("Endpoint: %s %s\n", stat.Method, stat.URL))
		sb.WriteString("------------------------\n")
		sb.WriteString(fmt.Sprintf("Total Requests:    %d\n", stat.TotalRequests))
		sb.WriteString(fmt.Sprintf("Successful:        %d\n", stat.SuccessRequests))
		sb.WriteString(fmt.Sprintf("Failed:            %d\n", stat.FailedRequests))
		sb.WriteString(fmt.Sprintf("Requests/second:   %.2f\n\n", stat.RequestsPerSecond))
		sb.WriteString("Latency Statistics:\n")
		sb.WriteString(fmt.Sprintf("  Average:    %v\n", stat.AverageDuration))
		sb.WriteString(fmt.Sprintf("  Median:     %v\n", stat.MedianDuration))
		sb.WriteString(fmt.Sprintf("  Minimum:    %v\n", stat.MinDuration))
		sb.WriteString(fmt.Sprintf("  Maximum:    %v\n", stat.MaxDuration))
		sb.WriteString(fmt.Sprintf("  95th %%:     %v\n", stat.Percentile95))
		sb.WriteString(fmt.Sprintf("  99th %%:     %v\n\n", stat.Percentile99))

		sb.WriteString("\nStatus Code Distribution:\n")
		for code, count := range stat.StatusCodes {
			sb.WriteString(fmt.Sprintf("  %d: %d requests\n", code, count))
		}
		sb.WriteString(fmt.Sprintf("  2xx Responses: %d\n", stat.SuccessCodes))
		sb.WriteString(fmt.Sprintf("  4xx Responses: %d\n", stat.ClientErrors))
		sb.WriteString(fmt.Sprintf("  5xx Responses: %d\n", stat.ServerErrors))
		sb.WriteString("\n")
	}

	return sb.String()
}

func CalculateLoadTest(results []runner.LoadTestResult) *LoadTestStats {
	stats := &LoadTestStats{
		EndpointStats: make(map[string]LoadStats),
	}

	for _, result := range results {
		stepStats := Calculate(result.Results)
		avgLatency := calculateAverageLatency(stepStats)

		stats.Steps = append(stats.Steps, StepStatistics{
			UserCount:         result.UserCount,
			DataSize:          result.DataSize,
			AverageLatency:    avgLatency,
			RequestsPerSecond: calculateOverallRPS(stepStats),
			SuccessRate:       calculateOverallSuccessRate(stepStats),
			ErrorRate:         calculateOverallErrorRate(stepStats),
		})

		// Update aggregate stats
		stats.TotalRequests += countTotalRequests(stepStats)
		updateLatencyStats(stats, avgLatency)
	}

	return stats
}

// Helper functions for calculations
func calculateAverageLatency(stats *Statistics) time.Duration {
	var total time.Duration
	count := 0
	for _, es := range stats.EndpointStats {
		total += es.AverageDuration
		count++
	}
	if count == 0 {
		return 0
	}
	return total / time.Duration(count)
}

func calculateOverallRPS(stats *Statistics) float64 {
	var total float64
	for _, es := range stats.EndpointStats {
		total += es.RequestsPerSecond
	}
	return total
}

func calculateOverallSuccessRate(stats *Statistics) float64 {
	totalRequests := 0
	totalSuccess := 0
	for _, es := range stats.EndpointStats {
		totalRequests += es.TotalRequests
		totalSuccess += es.SuccessRequests
	}
	if totalRequests == 0 {
		return 0
	}
	return float64(totalSuccess) / float64(totalRequests) * 100
}

func calculateOverallErrorRate(stats *Statistics) float64 {
	return 100 - calculateOverallSuccessRate(stats)
}

func countTotalRequests(stats *Statistics) int {
	total := 0
	for _, es := range stats.EndpointStats {
		total += es.TotalRequests
	}
	return total
}

func updateLatencyStats(stats *LoadTestStats, latency time.Duration) {
	if stats.MinLatency == 0 || latency < stats.MinLatency {
		stats.MinLatency = latency
	}
	if latency > stats.MaxLatency {
		stats.MaxLatency = latency
	}
	// Update average using weighted average
	if stats.AverageLatency == 0 {
		stats.AverageLatency = latency
	} else {
		stats.AverageLatency = (stats.AverageLatency + latency) / 2
	}
}
