package config

const (
	DefaultThreadCount     = 1
	DefaultConnectionCount = 1
	DefaultRequestCount    = 1
	DefaultThresholdPct    = 10.0
	DefaultHistoryDir      = "test-history"
	DefaultReportDir       = "performance-reports"
)

type Defaults struct {
	ThreadCount     int
	ConnectionCount int
	RequestCount    int
	ThresholdPct    float64
	HistoryDir      string
	ReportDir       string
}

func GetDefaults() *Defaults {
	return &Defaults{
		ThreadCount:     DefaultThreadCount,
		ConnectionCount: DefaultConnectionCount,
		RequestCount:    DefaultRequestCount,
		ThresholdPct:    DefaultThresholdPct,
		HistoryDir:      DefaultHistoryDir,
		ReportDir:       DefaultReportDir,
	}
}
