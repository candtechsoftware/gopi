package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"percipio.com/gopi/lib/git"
	"percipio.com/gopi/lib/logger"
	"percipio.com/gopi/lib/stats"
)

const (
	defaultHistoryDir  = "test-history"
	summaryFile        = "summary.json"
	perfHistoryDir     = "test-history/performance"
	userLoadHistoryDir = "test-history/user-load"
	dataLoadHistoryDir = "test-history/data-load"
)

type Store struct {
	baseDir      string
	thresholdPct float64
	gitInfo      GitMetadata
}

func NewStore(baseDir string, thresholdPct float64, useGit bool) (*Store, error) {
	var gitInfo GitMetadata

	if useGit {
		commitInfo, err := git.GetCommitInfo(useGit)
		if err != nil {
			logger.Warn("Git information not available: %v. Using timestamp-based tracking.", err)
			gitInfo = createTimestampBasedMetadata()
		} else {
			gitInfo = GitMetadata{
				CommitHash: commitInfo.Hash,
				ShortHash:  commitInfo.ShortHash,
				Timestamp:  commitInfo.Timestamp,
			}
		}
	} else {
		gitInfo = createTimestampBasedMetadata()
	}

	if baseDir == "" {
		baseDir = defaultHistoryDir
	}

	// Create history directories
	for _, dir := range []string{baseDir, perfHistoryDir, userLoadHistoryDir, dataLoadHistoryDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return &Store{
		baseDir:      baseDir,
		thresholdPct: thresholdPct,
		gitInfo:      gitInfo,
	}, nil
}

func createTimestampBasedMetadata() GitMetadata {
	now := time.Now()
	timestamp := now.Format("20060102-150405")
	return GitMetadata{
		CommitHash:    fmt.Sprintf("ts_%s", timestamp),
		ShortHash:     timestamp,
		Timestamp:     now,
		Branch:        "timestamp",
		CommitMessage: "Timestamp-based test run",
	}
}

func (s *Store) SaveResults(stats *stats.Statistics) (*TestHistory, error) {
	if err := os.MkdirAll(s.baseDir, 0755); err != nil {
		return nil, err
	}

	history := &TestHistory{
		RunID:        time.Now().Format("20060102-150405"),
		Timestamp:    time.Now(),
		Statistics:   stats,
		Endpoints:    make(map[string]*Comparison),
		ThresholdPct: s.thresholdPct,
		GitInfo:      s.gitInfo,
	}

	previous, err := s.LoadLatest()
	if err == nil && previous != nil {
		history.BaselineID = previous.RunID
		history.Degradation = s.compareWithBaseline(history, previous)
	}

	filename := filepath.Join(s.baseDir, history.RunID+".json")
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return nil, err
	}

	summary := &Summary{
		EndpointHistory: make(map[string][]TrendReport),
		Trends:          make(map[string]TrendReport),
	}

	data, err = os.ReadFile(filepath.Join(s.baseDir, summaryFile))
	if err == nil {
		if err := json.Unmarshal(data, summary); err != nil {
			return nil, err
		}
	}

	for endpoint, stats := range history.Statistics.EndpointStats {
		errorRate := float64(stats.FailedRequests) / float64(stats.TotalRequests) * 100
		trend := TrendReport{
			CommitHash:     s.gitInfo.CommitHash,
			CommitTime:     s.gitInfo.Timestamp,
			IterationMS:    float64(stats.AverageDuration.Milliseconds()),
			TotalRequests:  stats.TotalRequests,
			AvgLatencyMS:   float64(stats.AverageDuration.Milliseconds()),
			P50LatencyMS:   float64(stats.P50Latency.Milliseconds()),
			P95LatencyMS:   float64(stats.P95Latency.Milliseconds()),
			P99LatencyMS:   float64(stats.P99Latency.Milliseconds()),
			RPS:            stats.RequestsPerSecond,
			ErrorRateTrend: errorRate,
		}

		logger.Info("Saved trend for endpoint %s: avg=%.2f ms, p50=%.2f ms, p95=%.2f ms, p99=%.2f ms, reqs=%d\n",
			endpoint, trend.AvgLatencyMS, trend.P50LatencyMS, trend.P95LatencyMS, trend.P99LatencyMS, trend.TotalRequests)

		if _, exists := summary.EndpointHistory[endpoint]; !exists {
			summary.EndpointHistory[endpoint] = make([]TrendReport, 0)
		}
		summary.EndpointHistory[endpoint] = append(summary.EndpointHistory[endpoint], trend)
		summary.Trends[endpoint] = trend

		logger.Info("Saved trend for endpoint %s: ms=%.2f, reqs=%d\n",
			endpoint, trend.AvgLatencyMS, trend.TotalRequests)
	}

	data, err = json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return nil, err
	}

	if err := s.updateSummary(history); err != nil {
		logger.Error("Failed to update summary: %v", err)
		// Continue with the rest of the process but maybe
		// we should return an error here?
	}

	return history, os.WriteFile(filepath.Join(s.baseDir, summaryFile), data, 0644)
}

func (s *Store) LoadLatest() (*TestHistory, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" && entry.Name() != summaryFile {
			files = append(files, entry.Name())
		}
	}

	if len(files) == 0 {
		return nil, nil
	}

	sort.Strings(files)
	latest := files[len(files)-1]

	data, err := os.ReadFile(filepath.Join(s.baseDir, latest))
	if err != nil {
		return nil, err
	}

	var history TestHistory
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, err
	}

	return &history, nil
}

func (s *Store) compareWithBaseline(current, baseline *TestHistory) bool {
	hasDegradation := false

	for endpoint, currentStats := range current.Statistics.EndpointStats {
		if baselineStats, exists := baseline.Statistics.EndpointStats[endpoint]; exists {
			comparison := &Comparison{
				Current:  currentStats,
				Previous: baselineStats,
			}

			changes := DegradationReport{
				LatencyIncrease:     percentageIncrease(currentStats.AverageDuration.Seconds(), baselineStats.AverageDuration.Seconds()),
				ErrorRateIncrease:   percentageIncrease(float64(currentStats.FailedRequests), float64(baselineStats.FailedRequests)),
				ThroughputDecrease:  percentageDecrease(currentStats.RequestsPerSecond, baselineStats.RequestsPerSecond),
				SuccessRateDecrease: percentageDecrease(successRate(currentStats), successRate(baselineStats)),
			}

			comparison.Changes = changes
			comparison.Degradation = s.isDegraded(changes)
			current.Endpoints[endpoint] = comparison

			if comparison.Degradation {
				hasDegradation = true
			}
		}
	}

	return hasDegradation
}

func (s *Store) isDegraded(changes DegradationReport) bool {
	return changes.LatencyIncrease > s.thresholdPct ||
		changes.ErrorRateIncrease > s.thresholdPct ||
		changes.ThroughputDecrease > s.thresholdPct ||
		changes.SuccessRateDecrease > s.thresholdPct
}

func successRate(stats *stats.EndpointStatistics) float64 {
	if stats.TotalRequests == 0 {
		return 0
	}
	return float64(stats.SuccessRequests) / float64(stats.TotalRequests) * 100
}

func percentageIncrease(current, previous float64) float64 {
	if previous == 0 {
		return 0
	}
	return ((current - previous) / previous) * 100
}

func percentageDecrease(current, previous float64) float64 {
	return -percentageIncrease(current, previous)
}

type Summary struct {
	LastRun         time.Time                `json:"lastRun"`
	RunCount        int                      `json:"runCount"`
	Degradation     bool                     `json:"degradation"`
	History         []string                 `json:"history"`
	Trends          map[string]TrendReport   `json:"trends"`
	EndpointHistory map[string][]TrendReport `json:"endpointHistory"`
}

func (s *Store) updateSummary(current *TestHistory) error {
	logger.Info("Updating performance summary for run %s", current.RunID)
	summaryPath := filepath.Join(s.baseDir, summaryFile)
	var summary Summary

	data, err := os.ReadFile(summaryPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read summary: %w", err)
	}

	if err == nil {
		if err := json.Unmarshal(data, &summary); err != nil {
			return fmt.Errorf("failed to parse summary: %w", err)
		}
	}

	summary.LastRun = current.Timestamp
	summary.RunCount++
	summary.History = append(summary.History, current.RunID)
	summary.Degradation = current.Degradation

	if summary.EndpointHistory == nil {
		summary.EndpointHistory = make(map[string][]TrendReport)
	}
	if summary.Trends == nil {
		summary.Trends = make(map[string]TrendReport)
	}

	for endpoint, comparison := range current.Endpoints {
		trend := TrendReport{
			CommitHash:    s.gitInfo.CommitHash,
			CommitTime:    s.gitInfo.Timestamp,
			IterationMS:   float64(comparison.Current.AverageDuration.Milliseconds()),
			TotalRequests: comparison.Current.TotalRequests,
			AvgLatencyMS:  float64(comparison.Current.AverageDuration.Milliseconds()),
			RPS:           comparison.Current.RequestsPerSecond,
			P50LatencyMS:  float64(comparison.Current.P50Latency.Milliseconds()),
			P95LatencyMS:  float64(comparison.Current.P95Latency.Milliseconds()),
			P99LatencyMS:  float64(comparison.Current.P99Latency.Milliseconds()),
		}

		logger.Debug("Adding history point: endpoint=%s, hash=%s, ms=%.2f\n",
			endpoint, trend.CommitHash[:8], trend.AvgLatencyMS)

		if _, exists := summary.EndpointHistory[endpoint]; !exists {
			summary.EndpointHistory[endpoint] = make([]TrendReport, 0)
		}
		summary.EndpointHistory[endpoint] = append(summary.EndpointHistory[endpoint], trend)

		summary.Trends[endpoint] = trend
	}

	data, err = json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}

	for endpoint, history := range summary.EndpointHistory {
		logger.Info("Endpoint %s has %d history points\n", endpoint, len(history))
	}

	return os.WriteFile(summaryPath, data, 0644)
}

func (s *Store) GetSummary() (*Summary, error) {
	summaryPath := filepath.Join(s.baseDir, summaryFile)
	var summary Summary

	data, err := os.ReadFile(summaryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Summary{
				Trends: make(map[string]TrendReport),
			}, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, &summary); err != nil {
		return nil, err
	}

	return &summary, nil
}

func (s *Store) SaveLoadTestResults(stats *stats.LoadTestStats, testType string) (*LoadTestHistory, error) {
	var historyDir string
	switch testType {
	case TestTypeLoadUser:
		historyDir = userLoadHistoryDir
	case TestTypeLoadData:
		historyDir = dataLoadHistoryDir
	default:
		return nil, fmt.Errorf("invalid test type: %s", testType)
	}

	if err := os.MkdirAll(historyDir, 0755); err != nil {
		return nil, err
	}

	history := &LoadTestHistory{
		RunID:      time.Now().Format("20060102-150405"),
		Timestamp:  time.Now(),
		TestType:   testType,
		Statistics: stats,
		GitInfo:    s.gitInfo,
	}

	filename := filepath.Join(historyDir, history.RunID+".json")
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return nil, err
	}

	return history, os.WriteFile(filename, data, 0644)
}

// Add more methods for loading and comparing load test results...
