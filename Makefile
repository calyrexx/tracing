.PHONY: init start mocks lint test checks stop compush stage
MAKEFLAGS += --no-print-directory
GIT_BRANCH := $(shell git branch --show-current)

CHECK_EMOJI := ✅
ERROR_EMOJI := ❌
INFO_EMOJI := ℹ️
ARROW_UP := ⬆️

help:
	@echo "Available commands:"
	@echo "  build      - Build the binary file"
	@echo "  init       - Initialize the project"
	@echo "  start      - Start the application"
	@echo "  mocks      - Generate mocks"
	@echo "  lint       - Run linters"
	@echo "  test       - Run tests"
	@echo "  checks     - Run all checks (mocks, lint, test)"
	@echo "  stop       - Stop containers"
	@echo "  compush    - Run pre-commit checks, commit, and push changes"
	@echo "  stage      - Stage changes and push to Git"

build:
	@echo "$(INFO_EMOJI) Building the binary file..."
	@GOOS=linux CGO_ENABLED=0 go build -o portal ./cmd/main.go
	@echo "$(CHECK_EMOJI) Project built successfully!"

init:
	@echo "$(INFO_EMOJI) Initializing project..."
	@docker-compose up -d
	@echo "$(CHECK_EMOJI) Project initialized successfully!"

start:
	@echo "$(INFO_EMOJI) Starting the application..."
	@docker-compose up -d
	@trap 'exit 0' SIGINT; go run $(CURDIR)/cmd/main.go
	@echo "$(CHECK_EMOJI) Application started successfully!"

mocks:
	@echo "$(INFO_EMOJI) Generating mocks..."
	@(go generate ./... > mocks.log 2>&1 || (cat mocks.log && echo "$(ERROR_EMOJI) Error generating mocks! Check logs $(ARROW_UP)" && exit 1))
	@rm -f mocks.log
	@echo "$(CHECK_EMOJI) All mocks were generated successfully!"

lint:
	@echo "$(INFO_EMOJI) Running linters..."
	@(golangci-lint run ./... > lint.log 2>&1 || (cat lint.log && echo "$(ERROR_EMOJI) Linter found issues! Check logs $(ARROW_UP)" && exit 1))
	@rm -f lint.log
	@echo "$(CHECK_EMOJI) No lint errors found!"

test:
	@echo "$(INFO_EMOJI) Running tests..."
	@(go test ./... > test_output.log 2>&1 || (cat test_output.log && echo "$(ERROR_EMOJI) Tests failed! Check logs $(ARROW_UP)" && exit 1))
	@rm -f test_output.log
	@echo "$(CHECK_EMOJI) All tests PASSED!"

checks: mocks lint test
	@echo "$(CHECK_EMOJI) All checks completed successfully!"

stop:
	@echo "$(INFO_EMOJI) Stopping containers..."
	@docker-compose down
	@echo "$(CHECK_EMOJI) Containers stopped successfully!"

compush:
	@echo "$(INFO_EMOJI) Running pre-commit checks..."
	@$(MAKE) checks
	@echo "$(CHECK_EMOJI) Pre-commit checks passed! Moving to staging..."
	@$(MAKE) stage

stage:
	@echo "$(INFO_EMOJI) Staging changes..."
	@git add .
	@git commit -m "$(m)"
	@git push origin $(GIT_BRANCH)
	@echo "$(CHECK_EMOJI) Changes pushed to $(GIT_BRANCH)!"