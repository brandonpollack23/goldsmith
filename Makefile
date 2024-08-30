# Variables
CMD_DIR := ./cmd/goldsmith
PKG_DIR := ./pkg

# Default target
.PHONY: all
all: build

# Build the main application
.PHONY: build
build:
	go build -o bin/goldsmith $(CMD_DIR)

# Run the main application
.PHONY: run
run:
	go run $(CMD_DIR)

# Test all packages
.PHONY: test
test:
	go test ./...

# Clean up generated files
.PHONY: clean
clean:
	go clean
	rm -f bin

# Tidy go modules
.PHONY: tidy
tidy:
	find . -name "go.mod" -execdir go mod tidy -v \;

# Format the code
.PHONY: fmt
fmt:
	gofumpt -w ./...

# Docs
.PHONY: doc
doc:
	pkgsite -open
