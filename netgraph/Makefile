# Makefile for building 'netgraph' and 'gendot'

# Binary names
BINARY_NETGRAPH = netgraph
BINARY_GENDOT   = gendot
BINARY_GENTOPO   = gentopo

# Build output directory
BUILD_DIR = bin

# Go commands
GO      = go
GOCMD   = $(GO)
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST  = $(GOCMD) test
GOGET   = $(GOCMD) get

.PHONY: all deps build build-netgraph build-gendot build-gentopo run run-netgraph run-gendot run-gentopo clean test

# Default target: install deps, then build both binaries
all: deps build

# Install dependencies
deps:
	$(GOGET) -v ./...

# Build both binaries
build: build-netgraph build-gendot build-gentopo

# Build the netgraph binary from netgraph.go
build-netgraph:
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NETGRAPH) netgraph.go

# Build the gendot binary from gendot.go in subdir
build-gendot:
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_GENDOT) ./gendot

# Build the gentopo binary from gentopo.go 
build-gentopo:
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_GENTOPO) gentopo.go

# Run 'netgraph' (by default)
run: run-netgraph

# Run netgraph
run-netgraph:
	./$(BUILD_DIR)/$(BINARY_NETGRAPH)

# Run gendot
run-gendot:
	./$(BUILD_DIR)/$(BINARY_GENDOT)

# Run gentopo
run-gentopo:
	./$(BUILD_DIR)/$(BINARY_GENTOPO)

# Clean up build artifacts
clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

# Test the code in all subpackages
test:
	$(GOTEST) ./...
