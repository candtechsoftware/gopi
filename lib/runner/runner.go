package runner

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"math/rand"

	"percipio.com/gopi/lib/logger"
)

type Runner struct {
	client       *http.Client
	tasks        []Task
	workerCount  int
	requestCount int
}

func NewRunner(threadCount, requestCount int) *Runner {
	transport := &http.Transport{
		MaxIdleConns:        threadCount,
		MaxIdleConnsPerHost: threadCount,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   time.Second * 30,
	}

	return &Runner{
		client:       client,
		workerCount:  threadCount,
		requestCount: requestCount,
	}
}

func (r *Runner) Run() []Result {
	logger.Info("Starting benchmark with %d threads and %d requests per endpoint", r.workerCount, r.requestCount)
	logger.Info("Total endpoints to test: %d", len(r.tasks))

	taskChan := make(chan Task)
	resultChan := make(chan Result)
	var wg sync.WaitGroup

	logger.Info("Launching %d worker goroutines", r.workerCount)
	for i := 0; i < r.workerCount; i++ {
		wg.Add(1)
		go r.worker(i, taskChan, resultChan, &wg)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	go func() {
		for _, task := range r.tasks {
			for i := 0; i < r.requestCount; i++ {
				taskChan <- task
			}
		}
		close(taskChan)
	}()

	totalRequests := len(r.tasks) * r.requestCount
	completedRequests := 0
	var results []Result

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			progress := float64(completedRequests) / float64(totalRequests) * 100
			logger.Info("Progress: %.1f%% (%d/%d requests completed)\r",
				progress, completedRequests, totalRequests)
		}
	}()

	for result := range resultChan {
		results = append(results, result)
		completedRequests++

		if result.Error != nil {
			logger.Error("Request to %s failed: %v", result.URL, result.Error)
		}
	}

	logger.Info("\nBenchmark completed. Total requests processed: %d", len(results))
	return results
}

func (r *Runner) worker(id int, tasks <-chan Task, results chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()
	logger.Info("Worker %d started", id)

	for task := range tasks {
		start := time.Now()
		req, err := http.NewRequest(task.Method, task.URL, nil)
		if err != nil {
			logger.Error("Worker %d: Error making request to %s: %v", id, task.URL, err)
			results <- Result{
				URL:    task.URL,
				Method: task.Method,
				Error:  err,
			}
			continue
		}

		for k, v := range task.Headers {
			req.Header.Add(k, v)
		}

		resp, err := r.client.Do(req)
		duration := time.Since(start)

		if err != nil {
			logger.Error("Worker %d: Request to %s failed: %v", id, task.URL, err)
			results <- Result{
				URL:      task.URL,
				Method:   task.Method,
				Duration: duration,
				Error:    err,
			}
			continue
		}

		logger.Info("Worker %d: %s %s - Status: %d, Duration: %v",
			id, task.Method, task.URL, resp.StatusCode, duration)

		results <- Result{
			URL:        task.URL,
			Method:     task.Method,
			StatusCode: resp.StatusCode,
			Duration:   duration,
		}
		resp.Body.Close()
	}

	logger.Info("Worker %d finished", id)
}

func (r *Runner) AddTask(task Task) {
	r.tasks = append(r.tasks, task)
}

func (r *Runner) RunUserLoadTest(config UserLoadConfig) []LoadTestResult {
	var results []LoadTestResult
	currentUsers := config.StartUsers
	totalSteps := (config.MaxUsers-config.StartUsers)/config.StepUsers + 1

	logger.Info("Starting load test with %d steps", totalSteps)

	for stepNumber := 0; currentUsers <= config.MaxUsers; stepNumber++ {
		logger.Info("\nStep %d/%d: Testing with %d concurrent users",
			stepNumber+1, totalSteps, currentUsers)

		ctx, cancel := context.WithTimeout(context.Background(), config.DurationPerStep)
		resultChan := make(chan Result, currentUsers*len(r.tasks))
		var activeUsers atomic.Int32
		var totalRequests atomic.Int32
		var wg sync.WaitGroup

		// Progress monitoring
		go func() {
			start := time.Now()
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					reqs := totalRequests.Load()
					active := activeUsers.Load()
					elapsed := time.Since(start).Seconds()
					rps := float64(reqs) / elapsed

					logger.Info("Progress - Active: %d users | Total reqs: %d | RPS: %.2f | Elapsed: %.0fs",
						active, reqs, rps, elapsed)
				}
			}
		}()

		// Launch users
		for i := 0; i < currentUsers; i++ {
			wg.Add(1)
			go func(userID int) {
				defer wg.Done()

				client := &http.Client{
					Transport: &http.Transport{
						MaxIdleConns:        1,
						MaxIdleConnsPerHost: 1,
						IdleConnTimeout:     30 * time.Second,
					},
					Timeout: 10 * time.Second,
				}

				activeUsers.Add(1)
				defer activeUsers.Add(-1)

				// Stagger start
				time.Sleep(time.Duration(userID*100) * time.Millisecond)

				for {
					select {
					case <-ctx.Done():
						return
					default:
						task := r.tasks[rand.Intn(len(r.tasks))]
						result := r.executeRequest(client, task, userID)

						select {
						case resultChan <- result:
							totalRequests.Add(1)
						default:
						}

						// Randomized think time between 100ms and 1s
						time.Sleep(time.Duration(100+rand.Intn(900)) * time.Millisecond)
					}
				}
			}(i)
		}

		// Wait for step duration
		<-ctx.Done()
		logger.Info("Step %d completed, collecting results...", stepNumber+1)

		// Collect results
		close(resultChan)
		stepResults := make([]Result, 0)
		for result := range resultChan {
			stepResults = append(stepResults, result)
		}

		results = append(results, LoadTestResult{
			UserCount:  currentUsers,
			Results:    stepResults,
			Timestamp:  time.Now(),
			StepNumber: stepNumber,
		})

		cancel()
		wg.Wait()

		// Prepare for next step
		if currentUsers < config.MaxUsers {
			logger.Info("Cooling down before next step (5 seconds)...")
			time.Sleep(5 * time.Second)
			currentUsers += config.StepUsers
		} else {
			break
		}
	}

	return results
}

func (r *Runner) RunDataLoadTest(config DataLoadConfig) []LoadTestResult {
	var results []LoadTestResult
	currentSize := config.InitialDataSize

	for step := 0; step < config.StepsCount && currentSize <= config.MaxDataSize; step++ {
		logger.Info("Testing with data size: %d records...", currentSize)

		// Adjust request count based on data size
		originalRequestCount := r.requestCount
		r.requestCount = calculateRequestCount(currentSize)

		testResults := r.Run()

		results = append(results, LoadTestResult{
			DataSize:  currentSize,
			Results:   testResults,
			Timestamp: time.Now(),
		})

		// Reset request count
		r.requestCount = originalRequestCount

		logger.Info("Simulating data growth...")
		currentSize = int(float64(currentSize) * config.DataSizeMultiplier)
		time.Sleep(2 * time.Second) // Cool down period
	}

	return results
}

// Helper function to scale request count based on data size
func calculateRequestCount(dataSize int) int {
	// Reduce request count for larger data sizes to prevent overload
	if dataSize < 1000 {
		return 100
	} else if dataSize < 10000 {
		return 50
	} else if dataSize < 100000 {
		return 20
	}
	return 10
}

func (r *Runner) executeRequest(client *http.Client, task Task, userID int) Result {
	start := time.Now()

	req, err := http.NewRequest(task.Method, task.URL, nil)
	if err != nil {
		return Result{
			URL:       task.URL,
			Method:    task.Method,
			Error:     err,
			ThreadID:  userID,
			StartTime: start,
			EndTime:   time.Now(),
		}
	}

	// Add headers
	for k, v := range task.Headers {
		req.Header.Add(k, v)
	}

	// Execute request
	resp, err := client.Do(req)
	now := time.Now()

	if err != nil {
		return Result{
			URL:       task.URL,
			Method:    task.Method,
			Error:     err,
			Duration:  now.Sub(start),
			ThreadID:  userID,
			StartTime: start,
			EndTime:   now,
		}
	}
	defer resp.Body.Close()

	return Result{
		URL:        task.URL,
		Method:     task.Method,
		StatusCode: resp.StatusCode,
		Duration:   now.Sub(start),
		ThreadID:   userID,
		StartTime:  start,
		EndTime:    now,
	}
}
