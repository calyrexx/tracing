.PHONY: mocks lint test checks compush stage
MAKEFLAGS += --no-print-directory
GIT_BRANCH := $(shell git branch --show-current)
GIT_REMOTE := git@github.com:Calyr3x/tracing.git

CHECK_EMOJI := ✅
ERROR_EMOJI := ❌
INFO_EMOJI := ℹ️
ARROW_UP := ⬆️

help:
	@echo "Available commands:"
	@echo "  mocks      - Generate mocks"
	@echo "  lint       - Run linters"
	@echo "  test       - Run tests"
	@echo "  checks     - Run all checks (mocks, lint, test)"
	@echo "  compush    - Run pre-commit checks, commit, and push changes"
	@echo "  stage      - Stage changes and push to Git"

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

compush:
	@echo "$(INFO_EMOJI) Running pre-commit checks..."
	@$(MAKE) checks
	@echo "$(CHECK_EMOJI) Pre-commit checks passed! Moving to staging..."
	@$(MAKE) stage

stage:
	@echo "$(INFO_EMOJI) Staging changes..."
	@git add .
	@git commit -m "$(m)"
	@git push $(GIT_REMOTE) $(GIT_BRANCH)
	@echo "$(CHECK_EMOJI) Changes pushed to $(GIT_BRANCH)!"