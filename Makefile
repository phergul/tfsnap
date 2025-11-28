.PHONY: build test clean install uninstall

BINARY_NAME=tfsnap

INSTALL_DIR := $(if $(GOBIN),$(GOBIN),$(shell go env GOPATH)/bin)

build:
	@echo "Building $(BINARY_NAME)..."
	go build -o bin/$(BINARY_NAME) main.go

test:
	@echo "Running tests..."
	go test -v ./...

clean:
	@echo "Cleaning..."
	go clean
	@rm -rf bin
	@echo "Clean complete"

install:
	@echo "Installing $(BINARY_NAME) with go install..."
	go install .
	@echo "Installed to $(INSTALL_DIR)"

uninstall:
	@echo "Uninstalling $(BINARY_NAME) from $(INSTALL_DIR)..."
	@rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Uninstalled successfully"
