package history

import (
	"time"

	"percipio.com/gopi/lib/stats"
)

type TestHistory struct {
	RunID        string                 `json:"runId"`
	Timestamp    time.Time              `json:"timestamp"`
	Statistics   *stats.Statistics      `json:"statistics"`
	Endpoints    map[string]*Comparison `json:"endpoints"`
	BaselineID   string                 `json:"baselineId,omitempty"`
	Degradation  bool                   `json:"degradation"`
	ThresholdPct float64                `json:"thresholdPct"`
	GitInfo      GitMetadata            `json:"gitInfo"`
}

type GitMetadata struct {
	CommitHash    string    `json:"commitHash"`
	CommitMessage string    `json:"commitMessage"`
	Branch        string    `json:"branch"`
	ShortHash     string    `json:"shortHash"`
	Timestamp     time.Time `json:"timestamp"`
}

type Comparison struct {
	Current     *stats.EndpointStatistics `json:"current"`
	Previous    *stats.EndpointStatistics `json:"previous,omitempty"`
	Degradation bool                      `json:"degradation"`
	Changes     DegradationReport         `json:"changes"`
}

type DegradationReport struct {
	LatencyIncrease     float64 `json:"latencyIncrease"`
	ErrorRateIncrease   float64 `json:"errorRateIncrease"`
	ThroughputDecrease  float64 `json:"throughputDecrease"`
	SuccessRateDecrease float64 `json:"successRateDecrease"`
}

// TrendReport represents performance metrics for an endpoint at a specific point in time
type TrendReport struct {
	CommitHash       string    `json:"commitHash"`
	CommitTime       time.Time `json:"commitTime"`
	IterationMS      float64   `json:"iterationMs"`
	TotalRequests    int       `json:"totalRequests"`
	AvgLatencyMS     float64   `json:"avgLatencyMs"`
	P50LatencyMS     float64   `json:"p50LatencyMs"`
	P95LatencyMS     float64   `json:"p95LatencyMs"`
	P99LatencyMS     float64   `json:"p99LatencyMs"`
	RPS              float64   `json:"rps"`
	ErrorRateTrend   float64   `json:"errorRateTrend"`
	TrendPercent     float64   `json:"trendPercent"`
	BaselineHash     string    `json:"baselineHash,omitempty"`
	LatencyTrend     float64   `json:"latencyTrend"`
	ThroughputTrend  float64   `json:"throughputTrend"`
	SuccessRateTrend float64   `json:"successRateTrend"`
	MedianLatencyMS  float64   `json:"medianLatencyMs"`
}

// Stats holds formatted statistics for display
type Stats struct {
	AvgLatency        string
	LatencyChange     string
	SuccessRate       string
	SuccessRateChange string
	RPS               string
	RPSChange         string
	TotalRequests     string
	ErrorRate         string
	ErrorRateChange   string
	P50Latency        string
	P95Latency        string
	P99Latency        string
}

type LoadTestHistory struct {
	RunID      string               `json:"runId"`
	Timestamp  time.Time            `json:"timestamp"`
	TestType   string               `json:"testType"` // "user" or "data"
	Statistics *stats.LoadTestStats `json:"statistics"`
	BaselineID string               `json:"baselineId,omitempty"`
	GitInfo    GitMetadata          `json:"gitInfo"`
	Steps      []LoadTestStep       `json:"steps"`
}

type LoadTestStep struct {
	UserCount  int                   `json:"userCount,omitempty"`
	DataSize   int                   `json:"dataSize,omitempty"`
	Statistics *stats.StepStatistics `json:"statistics"`
	Timestamp  time.Time             `json:"timestamp"`
}

const (
	TestTypePerf     = "performance"
	TestTypeLoadUser = "user-load"
	TestTypeLoadData = "data-load"
)
