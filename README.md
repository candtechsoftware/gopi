# GOPI - Go Performance and Load Testing Tool

A performance testing tool for HTTP APIs that tracks performance trends over time.

## Features

- HTTP endpoint performance testing
- Historical trend tracking
- Git integration for commit-based comparisons
- Visual performance graphs
- Configurable concurrency levels

## Prerequisites

- Go 1.20 or higher
- Make (optional, for using Makefile commands)

## Installation

```bash
go install github.com/percipio/gopi@latest
```

## Usage

GOPI supports three testing modes:
- Standard Performance Test (`--test-perf`)
- User Load Test (`--test-load-user`)
- Data Load Test (`--test-load-data`)

### Basic Configuration

Create a JSON file with your endpoints:

```json
[
  {
    "url": "https://api.example.com/users",
    "method": "GET",
    "headers": {
      "Authorization": "Bearer your-token"
    }
  }
]
```

### Example Commands

#### Standard Performance Test
```bash
# Basic performance test
gopi --file endpoints.json --test-perf

# Performance test with custom settings
gopi --file endpoints.json --test-perf \
  --thread-count 10 \
  --request-count 100 \
  --connection-count 5

# Performance test without git integration
gopi --file endpoints.json --test-perf --no-git
```

#### User Load Test
```bash
# Basic user load test
gopi --file endpoints.json --test-load-user

# User load test with custom concurrency settings
gopi --file endpoints.json --test-load-user \
  --start-users 5 \
  --max-users 100 \
  --step-users 10 \
  --step-duration 120

# Intensive user load test
gopi --file endpoints.json --test-load-user \
  --start-users 10 \
  --max-users 500 \
  --step-users 50 \
  --step-duration 300 \
  --request-count 1000 \
  --thread-count 50
```

#### Data Load Test
```bash
# Basic data load test
gopi --file endpoints.json --test-load-data

# Data load test with custom data volume settings
gopi --file endpoints.json --test-load-data \
  --initial-data 5000 \
  --max-data 500000 \
  --data-multiplier 10 \
  --data-steps 6

# Comprehensive data load test
gopi --file endpoints.json --test-load-data \
  --initial-data 10000 \
  --max-data 1000000 \
  --data-multiplier 5 \
  --data-steps 8 \
  --thread-count 20 \
  --request-count 200
```

### Common Options

| Flag | Description | Default |
|------|-------------|---------|
| `--file`, `-f` | JSON file containing endpoints | Required |
| `--thread-count`, `-tc` | Number of threads | 1 |
| `--connection-count`, `-cc` | Number of connections | 1 |
| `--request-count`, `-rc` | Requests per endpoint | 1 |
| `--no-git` | Disable git integration | false |

### User Load Test Options

| Flag | Description | Default |
|------|-------------|---------|
| `--start-users` | Initial number of users | 2 |
| `--max-users` | Maximum number of users | 50 |
| `--step-users` | Users to add per step | 5 |
| `--step-duration` | Duration per step (seconds) | 60 |

### Data Load Test Options

| Flag | Description | Default |
|------|-------------|---------|
| `--initial-data` | Initial data size | 1000 |
| `--max-data` | Maximum data size | 100000 |
| `--data-multiplier` | Growth multiplier | 5.0 |
| `--data-steps` | Number of test steps | 4 |

### Running Tests

Using Make:
```bash
# Build and run
make run ARGS="-f examples/endpoints.json -tc 10 -rc 100"

# Build optimized release version
make release

# View latest test results
make view-report
```

Direct execution:
```bash
./bin/gopi -f endpoints.json --thread-count 10 --request-count 100
```

### Version Control Integration

The tool automatically detects the execution environment and handles version information appropriately:

1. **GitHub Actions:**
   - Automatically uses GitHub environment variables
   - Tracks commits, branches, and repository information
   - Perfect for CI/CD performance monitoring

2. **Local Git Repository:**
   - Uses local git commands to track commits
   - Maintains history across test runs
   - Compares performance between commits

3. **Non-Git Environment:**
   - Uses `--no-git` flag for timestamp-based tracking
   - Generates consistent hashes from timestamps
   - Maintains comparable history without git

## Output and Reports

Tests generate:
- Real-time progress output
- Performance metrics
- JSON reports in `test-history/`
- Visual graphs in `performance-reports/`

### Performance History
Test results are stored in the `test-history` directory:
- Individual test results
- Historical trends
- Performance comparisons
- Degradation analysis

## Project Structure

```
.
├── cmd/
│   └── gopi/    # Main application entry point
├── lib/
│   ├── app/               # Application logic
│   ├── config/            # Configuration handling
│   ├── history/           # Historical data management
│   ├── runner/            # Test execution engine
│   ├── stats/             # Statistics calculation
│   └── viz/               # Visualization generation
├── examples/              # Example configurations
├── performance-reports/   # Generated test reports (in .gitignore for this repo)
├── test-history/         # Historical test data (in .gitignore for this repo)
├── Makefile
└── README.md
```

## Make Commands

```bash
make build           # Build the application
make release         # Build optimized release version
make run            # Run the application (requires ARGS)
make clean          # Clean build artifacts
make test           # Run tests
make fmt            # Format code
make view-report    # Open latest test report
```

## CI/CD Integration

Example GitHub Actions workflow:
```yaml
name: Performance Tests
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Performance Test
        run: gopi -f config.json --test-perf
      - name: User Load Test
        run: gopi -f config.json --test-load-user
      - name: Data Load Test
        run: gopi -f config.json --test-load-data
```
