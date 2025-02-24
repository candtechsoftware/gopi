package viz

import (
	_ "embed"
	"fmt"
	"html/template"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	hist "percipio.com/gopi/lib/history"
	"percipio.com/gopi/lib/logger"
	"percipio.com/gopi/lib/util"
)

//go:embed static/graph.js
var graphJS string

const (
	defaultPointLimit = 20
	fixedGraphWidth   = 1000.0
	xPadding          = 50.0
)

const htmlTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>Performance Test Results</title>
    <style>
        .graph { margin: 20px; }
        .metric { margin-bottom: 40px; }
        .line { fill: none; stroke-width: 2; }
        .point { fill: #fff; stroke-width: 2; }
        .latency { stroke: #ff6b6b; }
        .throughput { stroke: #4ecdc4; }
        .error { stroke: #fc5c65; }
        .success { stroke: #95a5a6; }
        .axis { stroke: #333; }
        .label { font-size: 12px; fill: #333; }
        .grid { stroke: #eee; stroke-width: 1; }
        .endpoint-selector {
            margin: 20px;
            padding: 10px;
        }
        .endpoint-selector select {
            padding: 5px;
            font-size: 16px;
            width: 400px;
        }
        .endpoint-graph {
            display: none;
        }
        .endpoint-graph.active {
            display: block;
        }
        .stats-panel {
            display: flex;
            flex-wrap: wrap;
            justify-content: space-between;
            margin: 20px;
            padding: 20px;
            background: #f8f9fa;
            border-radius: 4px;
            gap: 20px;
        }
        .stat-row {
            width: 100%;
            display: flex;
            justify-content: space-between;
            gap: 20px;
        }
        .stat-box {
            flex: 1;
            text-align: center;
            padding: 15px;
            background: white;
            border-radius: 8px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1);
        }
        .stat-label {
            font-size: 14px;
            color: #666;
            margin-bottom: 5px;
        }
        .stat-value {
            font-size: 24px;
            font-weight: bold;
            color: #333;
        }
        .stat-unit {
            font-size: 12px;
            color: #666;
        }
        .commit-label {
            font-size: 12px;
            fill: #333;
            text-anchor: middle;
            dominant-baseline: hanging;
            writing-mode: horizontal-tb; 
            transform: none; 
        }
        .trend-info {
            margin: 10px 0;
            font-size: 14px;
        }
        .trend-label {
            color: #666;
        }
        .trend-value {
            margin-left: 10px;
            font-weight: bold;
        }
        .trend-up {
            color: #ff6b6b;
        }
        .trend-down {
            color: #4ecdc4;
        }
        .trend-line {
            stroke-width: 2;
            stroke-dasharray: 5,5;
        }
        .change-indicator {
            font-size: 14px;
            margin-left: 5px;
        }
        .change-positive {
            color: #ff6b6b;
        }
        .change-negative {
            color: #4ecdc4;
        }
        .connection-line {
            fill: none;
            stroke: #ff6b6b;
            stroke-width: 2;
        }
        .point-limit-selector {
            margin: 20px;
            padding: 10px;
        }
        .point-limit-selector select {
            padding: 5px;
            font-size: 14px;
            margin-left: 10px;
        }
        .graph-container {
            width: 100%;
            margin: 20px;
        }
        .graph {
            width: 100%;
            height: 450px;
        }
        .point-group, .label-group, .lines-container {
            transition: all 0.3s ease;
        }
        .connection-line {
            fill: none;
            stroke: #ff6b6b;
            stroke-width: 2;
            transition: all 0.3s ease;
        }
    </style>
</head>
<body>
    <h1>Performance Test Results</h1>
    <div class="endpoint-selector">
        <select id="endpointSelect" onchange="showEndpoint(this.value)">
            <option value="">Select an endpoint</option>
            {{range $key, $value := .Trends}}
            <option value="{{$key}}">{{$key}}</option>
            {{end}}
        </select>
    </div>
    
    <div class="point-limit-selector">
        <label>Show points: </label>
        <select id="pointLimit" onchange="updatePointLimit(this.value)">
            <option value="10">Last 10</option>
            <option value="20" selected>Last 20</option>
            <option value="30">Last 30</option>
            <option value="0">All</option>
        </select>
    </div>

    {{range $key, $value := .Trends}}
    <div id="{{$key}}" class="endpoint-graph">
        <div class="stats-panel">
            <div class="stat-row">
                <div class="stat-box">
                    <div class="stat-label">Average Response Time</div>
                    <div class="stat-value">
                        {{$value.Stats.AvgLatency}}
                        <span class="change-indicator {{if isPositive $value.Stats.LatencyChange}}change-positive{{else}}change-negative{{end}}">
                            {{$value.Stats.LatencyChange}}
                        </span>
                    </div>
                    <div class="stat-unit">ms</div>
                </div>
                <div class="stat-box">
                    <div class="stat-label">Success Rate</div>
                    <div class="stat-value">
                        {{$value.Stats.SuccessRate}}
                        <span class="change-indicator {{if isPositive $value.Stats.SuccessRateChange}}change-positive{{else}}change-negative{{end}}">
                            {{$value.Stats.SuccessRateChange}}
                        </span>
                    </div>
                    <div class="stat-unit">%</div>
                </div>
                <div class="stat-box">
                    <div class="stat-label">Requests/Second</div>
                    <div class="stat-value">
                        {{$value.Stats.RPS}}
                        <span class="change-indicator {{if isPositive $value.Stats.RPSChange}}change-positive{{else}}change-negative{{end}}">
                            {{$value.Stats.RPSChange}}
                        </span>
                    </div>
                    <div class="stat-unit">req/s</div>
                </div>
                <div class="stat-box">
                    <div class="stat-label">Total Requests</div>
                    <div class="stat-value">{{$value.Stats.TotalRequests}}</div>
                    <div class="stat-unit">requests</div>
                </div>
            </div>
            <div class="stat-row">
                <div class="stat-box">
                    <div class="stat-label">Error Rate</div>
                    <div class="stat-value">
                        {{$value.Stats.ErrorRate}}
                        <span class="change-indicator {{if isPositive $value.Stats.ErrorRateChange}}change-positive{{else}}change-negative{{end}}">
                            {{$value.Stats.ErrorRateChange}}
                        </span>
                    </div>
                    <div class="stat-unit">%</div>
                </div>
                <div class="stat-box">
                    <div class="stat-label">P50 Latency</div>
                    <div class="stat-value">{{$value.Stats.P50Latency}}</div>
                    <div class="stat-unit">ms</div>
                </div>
                <div class="stat-box">
                    <div class="stat-label">P95 Latency</div>
                    <div class="stat-value">{{$value.Stats.P95Latency}}</div>
                    <div class="stat-unit">ms</div>
                </div>
                <div class="stat-box">
                    <div class="stat-label">P99 Latency</div>
                    <div class="stat-value">{{$value.Stats.P99Latency}}</div>
                    <div class="stat-unit">ms</div>
                </div>
            </div>
        </div>

        <div class="metric">
            <h3>Performance Trend (ms/iter)</h3>
            <div class="trend-info">
                <span class="trend-label">Baseline Commit: {{$value.BaselineHash}}</span>
                <span class="trend-value {{if isPositive $value.TrendPercent}}trend-up{{else}}trend-down{{end}}">
                    {{printf "%.2f%%" $value.TrendPercent}}
                </span>
            </div>
            <div class="graph-container">
                <svg viewBox="0 0 1200 450" preserveAspectRatio="xMidYMid meet" class="graph">
                    <g transform="translate(50, 20)">
                        <!-- Y Axis -->
                        <line x1="0" y1="0" x2="0" y2="300" class="axis"/>
                        {{range $value.YAxisLabels}}
                        <text x="-40" y="{{.Y}}" class="label">{{.Value}} ms/iter</text>
                        {{end}}

                        <!-- Graph Content -->
                        <g id="graphContent">
                            <line x1="0" y1="300" x2="1100" y2="300" class="axis"/>
                            
                            <g class="lines-container">
                                <path d="{{$value.ConnectionPath}}" class="connection-line" />
                            </g>
                            
                            {{range $i, $p := $value.Points}}
                            <g class="point-group" data-index="{{$i}}">
                                <circle cx="{{$p.X}}" cy="{{$p.Y}}" r="4" class="point latency"/>
                            </g>
                            {{end}}

                            {{range $i, $l := $value.XAxisLabels}}
                            <g class="label-group" data-index="{{$i}}">
                                <text x="{{$l.X}}" y="340" class="commit-label" title="{{$l.Title}}">{{$l.Label}}</text>
                            </g>
                            {{end}}

                            <line 
                                x1="0" y1="{{$value.BaselineY}}" 
                                x2="1100" y2="{{$value.CurrentY}}"
                                class="trend-line {{if isPositive $value.TrendPercent}}trend-up{{else}}trend-down{{end}}"
                            />
                        </g>
                    </g>
                </svg>
            </div>
        </div>
    </div>
    {{end}}

    <script>
        {{.JavaScript}}
    </script>
</body>
</html>`

type GraphData struct {
	Trends      map[string]TrendGraph
	TotalPoints int
	JavaScript  template.JS
}

type TrendGraph struct {
	YAxisLabels    []AxisLabel
	XAxisLabels    []AxisLabel
	LatencyPoints  []Point
	LatencyPath    template.HTML
	ThroughputPath template.HTML
	ErrorPath      template.HTML
	SuccessPath    template.HTML
	Stats          hist.Stats
	Points         []Point
	BaselineHash   string
	TrendPercent   float64
	BaselineY      float64
	CurrentY       float64
	ConnectionPath string
	TotalPoints    int
	VisiblePoints  int
}

type AxisLabel struct {
	X     float64
	Y     float64
	Label string
	Value float64
	Title string
}

type Point struct {
	X     float64
	Y     float64
	Value float64
	Label string
}

func GenerateGraph(summary *hist.Summary, outputDir string) (string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", err
	}

	data := &GraphData{
		Trends:     make(map[string]TrendGraph),
		JavaScript: template.JS(graphJS),
	}

	maxPoints := 0
	for endpoint, trend := range summary.Trends {
		logger.Info("Processing endpoint %s with trend data: ms=%.2f, reqs=%d\n",
			endpoint, trend.AvgLatencyMS, trend.TotalRequests)

		history := summary.EndpointHistory[endpoint]
		data.Trends[endpoint] = generateEndpointGraph(trend, history)
		if len(history) > maxPoints {
			maxPoints = len(history)
		}
	}
	data.TotalPoints = maxPoints

	funcMap := template.FuncMap{
		"toFloat64": func(v interface{}) float64 {
			switch value := v.(type) {
			case int:
				return float64(value)
			case int64:
				return float64(value)
			case float32:
				return float64(value)
			case float64:
				return value
			case string:
				f, _ := strconv.ParseFloat(value, 64)
				return f
			default:
				return 0
			}
		},
		"isPositive": func(v interface{}) bool {
			f := toFloat64(v)
			return f > 0
		},
	}

	tmpl, err := template.New("graph").Funcs(funcMap).Parse(strings.Replace(htmlTemplate,
		`{{if gt (toFloat64 $data.TrendPercent) 0}}`,
		`{{if isPositive $data.TrendPercent}}`,
		-1))
	if err != nil {
		return "", err
	}

	outputFile := filepath.Join(outputDir, fmt.Sprintf("performance_%s.html",
		time.Now().Format("20060102_150405")))
	f, err := os.Create(outputFile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return "", err
	}

	return outputFile, nil
}

func percentageChange(current, previous float64) float64 {
	if previous == 0 {
		return 0
	}
	return ((current - previous) / previous) * 100
}

func generateEndpointGraph(t hist.TrendReport, history []hist.TrendReport) TrendGraph {
	graph := TrendGraph{}

	points := make([]hist.TrendReport, 0, len(history)+1)
	points = append(points, history...)
	if len(points) == 0 || points[len(points)-1].CommitHash != t.CommitHash {
		points = append(points, t)
	}

	graph.TotalPoints = len(points)

	var maxMs float64
	for _, h := range points {
		if h.AvgLatencyMS > maxMs {
			maxMs = h.AvgLatencyMS
		}
	}
	maxMs = math.Ceil(maxMs * 1.2)

	for i := 0; i <= 5; i++ {
		value := (float64(i) * maxMs) / 5.0
		y := 300.0 * (1.0 - value/maxMs)
		graph.YAxisLabels = append(graph.YAxisLabels, AxisLabel{
			Y:     y,
			Value: value,
		})
	}

	spacing := fixedGraphWidth
	if len(points) > 1 {
		spacing = fixedGraphWidth / float64(len(points)-1)
	}

	for i, h := range points {
		x := xPadding + (float64(i) * spacing)
		y := scaleValue(h.AvgLatencyMS, 0, maxMs, 300, 0)

		logger.Debug("Point %d: hash=%s, x=%.1f, y=%.1f, ms=%.2f\n",
			i, h.CommitHash[:8], x, y, h.AvgLatencyMS)

		graph.Points = append(graph.Points, Point{
			X:     x,
			Y:     y,
			Value: h.AvgLatencyMS,
		})

		graph.XAxisLabels = append(graph.XAxisLabels, AxisLabel{
			X:     x,
			Label: h.CommitHash[:7],
			Title: fmt.Sprintf("%s\n%s", h.CommitHash, h.CommitTime.Format("2006-01-02 15:04:05")),
		})
	}

	var pathBuilder strings.Builder
	for i, p := range graph.Points {
		if i == 0 {
			pathBuilder.WriteString(fmt.Sprintf("M %f %f", p.X, p.Y))
		} else {
			pathBuilder.WriteString(fmt.Sprintf(" L %f %f", p.X, p.Y))
		}
	}
	graph.ConnectionPath = pathBuilder.String()

	var changes struct {
		latency     float64
		rps         float64
		successRate float64
		errorRate   float64
	}

	if len(points) > 1 {
		baseline := points[0]
		changes.latency = t.AvgLatencyMS - baseline.AvgLatencyMS
		changes.rps = t.RPS - baseline.RPS
		changes.successRate = (100.0 - t.ErrorRateTrend) - (100.0 - baseline.ErrorRateTrend)
		changes.errorRate = t.ErrorRateTrend - baseline.ErrorRateTrend
	}

	graph.Stats = hist.Stats{
		AvgLatency:        util.FormatFloat(t.AvgLatencyMS),
		LatencyChange:     util.FormatChange(changes.latency),
		SuccessRate:       util.FormatFloat(100.0 - t.ErrorRateTrend), // Fixed: use ErrorRateTrend
		SuccessRateChange: util.FormatChange(changes.successRate),
		RPS:               util.FormatFloat(t.RPS),
		RPSChange:         util.FormatChange(changes.rps),
		TotalRequests:     fmt.Sprintf("%d", t.TotalRequests),
		ErrorRate:         util.FormatFloat(t.ErrorRateTrend), // Fixed: use ErrorRateTrend
		ErrorRateChange:   util.FormatChange(changes.errorRate),
		P50Latency:        util.FormatFloat(t.P50LatencyMS),
		P95Latency:        util.FormatFloat(t.P95LatencyMS),
		P99Latency:        util.FormatFloat(t.P99LatencyMS),
	}

	if len(points) > 1 {
		firstPoint := points[0]
		lastPoint := points[len(points)-1]
		graph.BaselineHash = firstPoint.CommitHash[:7]
		graph.TrendPercent = percentageChange(lastPoint.IterationMS, firstPoint.IterationMS)
		graph.BaselineY = scaleValue(firstPoint.IterationMS, 0, maxMs, 300, 0)
		graph.CurrentY = scaleValue(lastPoint.IterationMS, 0, maxMs, 300, 0)
	}

	if len(points) == 1 {
		graph.Stats = hist.Stats{
			AvgLatency:    fmt.Sprintf("%.2f", t.AvgLatencyMS),
			SuccessRate:   fmt.Sprintf("%.2f", 100.0-t.ErrorRateTrend),
			RPS:           fmt.Sprintf("%.2f", t.RPS),
			TotalRequests: fmt.Sprintf("%d", t.TotalRequests),
			ErrorRate:     fmt.Sprintf("%.2f", t.ErrorRateTrend),
		}
		graph.BaselineHash = t.CommitHash[:7]
	}

	return graph
}

func scaleValue(value, minInput, maxInput, minOutput, maxOutput float64) float64 {
	return (value-minInput)*(maxOutput-minOutput)/(maxInput-minInput) + minOutput
}

func toFloat64(v interface{}) float64 {
	switch value := v.(type) {
	case int:
		return float64(value)
	case int64:
		return float64(value)
	case float32:
		return float64(value)
	case float64:
		return value
	case string:
		f, _ := strconv.ParseFloat(value, 64)
		return f
	default:
		return 0
	}
}
