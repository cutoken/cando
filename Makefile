# Cando Build System
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -s -w"
BUILD_DIR := dist

.PHONY: all clean build build-linux build-darwin build-windows release dev install test fetch-openrouter-models fetch-model-contexts

all: clean build-linux build-darwin build-windows

# Fetch OpenRouter models at build time (sorted by weekly popularity)
fetch-openrouter-models:
	@echo "Fetching OpenRouter models (sorted by popularity)..."
	@curl -s 'https://openrouter.ai/api/frontend/models/find?order=top-weekly' | jq '[.data.models[] | select(.has_text_output == true) | {id: .endpoint.model_variant_slug, name: .name, capabilities: .input_modalities, pricing: {prompt: .endpoint.pricing.prompt, completion: .endpoint.pricing.completion}}]' > internal/agent/webui/openrouter-models.json
	@echo "OpenRouter models updated ($(shell jq length internal/agent/webui/openrouter-models.json) models)"

# Generate model context lengths JSON
fetch-model-contexts:
	@echo "Generating model contexts..."
	@curl -s 'https://openrouter.ai/api/frontend/models/find?order=top-weekly' | jq '[.data.models[] | select(.has_text_output == true) | {"key": ("openrouter/" + .endpoint.model_variant_slug), "value": .context_length}] | map({(.key): .value}) | add' > /tmp/or-contexts.json
	@echo '{"zai/glm-4.6": 200000, "zai/glm-4.5": 128000, "zai/glm-4.5-air": 128000}' | jq '.' > /tmp/zai-contexts.json
	@jq -s '.[0] * .[1]' /tmp/or-contexts.json /tmp/zai-contexts.json > internal/config/model-contexts.json
	@rm -f /tmp/or-contexts.json /tmp/zai-contexts.json
	@echo "Model contexts generated"

build: fetch-openrouter-models fetch-model-contexts
	@echo "Building for current platform..."
	@go build $(LDFLAGS) -o cando ./cmd/cando
	@echo "Built: ./cando"

dev:
	@if ! command -v air > /dev/null; then \
		echo "Installing air for live reload..."; \
		go install github.com/cosmtrek/air@latest; \
	fi
	@echo "Starting live reload server..."
	@air

clean:
	rm -rf $(BUILD_DIR)
	rm -f cando cando-* tmp/cando

build-linux:
	@echo "Building for Linux (amd64)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/cando-linux-amd64 ./cmd/cando
	@echo "Building for Linux (arm64)..."
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/cando-linux-arm64 ./cmd/cando

build-darwin:
	@echo "Building for macOS (amd64)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/cando-darwin-amd64 ./cmd/cando
	@echo "Building for macOS (arm64 - Apple Silicon)..."
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/cando-darwin-arm64 ./cmd/cando

build-windows:
	@echo "Building for Windows (amd64)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/cando-windows-amd64.exe ./cmd/cando

release: all
	@echo "Creating release archives..."
	@cd $(BUILD_DIR) && \
	for file in cando-*; do \
		if [[ "$$file" == *.exe ]]; then \
			zip "$${file%.exe}.zip" "$$file"; \
		else \
			tar czf "$$file.tar.gz" "$$file"; \
		fi; \
	done
	@echo "Release artifacts created in $(BUILD_DIR)/"

install:
	@echo "Installing to ~/.local/bin/cando..."
	@mkdir -p ~/.local/bin
	@go build $(LDFLAGS) -o ~/.local/bin/cando ./cmd/cando
	@chmod +x ~/.local/bin/cando
	@echo "Installed! Make sure ~/.local/bin is in your PATH"
	@echo 'Add to your shell rc: export PATH="$$HOME/.local/bin:$$PATH"'

test:
	go test -v ./...

.DEFAULT_GOAL := all
