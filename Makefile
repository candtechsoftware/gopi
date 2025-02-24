.PHONY: build clean test run release fmt view-report

BINARY_NAME=gopi
BUILD_DIR=bin
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Build flags
LDFLAGS=-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}
GOFLAGS=-trimpath

build:
	@echo "Building ${BINARY_NAME} (debug mode)..."
	@mkdir -p ${BUILD_DIR}
	@go build -ldflags "-X percipio.com/gopi/lib/logger.debugMode=true" \
		-o ${BUILD_DIR}/${BINARY_NAME} ./cmd/api-perf-tester

clean:
	@echo "Cleaning..."
	@rm -rf ${BUILD_DIR}
	@rm -rf test-history
	@rm -rf performance-reports
	@echo "Cleaned binary and generated directories"

test:
	@echo "Running tests..."
	@go test ./...

release:
	@echo "Building optimized release binary..."
	@mkdir -p ${BUILD_DIR}
	@go build \
		-ldflags "-s -w -X percipio.com/gopi/lib/logger.debugMode=false" \
		${GOFLAGS} \
		-o ${BUILD_DIR}/${BINARY_NAME} \
		./cmd/api-perf-tester

run: build
	@./${BUILD_DIR}/${BINARY_NAME} $(ARGS)

fmt:
	@echo "Formatting code..."
	@go fmt ./...

view-report:
	@latest=$$(ls -t performance-reports/performance_*.html | head -n1); \
	if [ -n "$$latest" ]; then \
		echo "Opening $$latest"; \
		open "$$latest" 2>/dev/null || xdg-open "$$latest" 2>/dev/null || start "$$latest" 2>/dev/null; \
	else \
		echo "No performance reports found"; \
	fi
