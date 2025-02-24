package runner

import (
	"time"
)

type Task struct {
	URL     string
	Method  string
	Headers map[string]string
	Body    []byte
}

type Result struct {
	URL        string
	Method     string
	StatusCode int
	Duration   time.Duration
	Error      error
	ThreadID   int
	StartTime  time.Time
	EndTime    time.Time
}

type UserLoadConfig struct {
	StartUsers      int
	MaxUsers        int
	StepUsers       int
	DurationPerStep time.Duration
}

type DataLoadConfig struct {
	InitialDataSize    int
	MaxDataSize        int
	DataSizeMultiplier float64
	StepsCount         int
}

type LoadTestResult struct {
	UserCount  int // For user load tests
	DataSize   int // For data load tests
	Results    []Result
	Timestamp  time.Time
	StepNumber int
}
