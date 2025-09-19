.PHONY: lint compush stage
MAKEFLAGS += --no-print-directory
GIT_BRANCH := $(shell git branch --show-current)
GIT_REMOTE := git@github.com:calyrexx/tracing.git

CHECK_EMOJI := ✅
ERROR_EMOJI := ❌
INFO_EMOJI := ℹ️
ARROW_UP := ⬆️

help:
	@echo "Available commands:"
	@echo "  lint       - Run linters"
	@echo "  compush    - Run pre-commit checks, commit, and push changes"
	@echo "  stage      - Stage changes and push to Git"

lint:
	@echo "$(INFO_EMOJI) Running linters..."
	@(golangci-lint run ./... > lint.log 2>&1 || (cat lint.log && echo "$(ERROR_EMOJI) Linter found issues! Check logs $(ARROW_UP)" && exit 1))
	@rm -f lint.log
	@echo "$(CHECK_EMOJI) No lint errors found!"

compush:
	@echo "$(INFO_EMOJI) Running pre-commit checks..."
	@$(MAKE) lint
	@echo "$(CHECK_EMOJI) Pre-commit checks passed! Moving to staging..."
	@$(MAKE) stage

stage:
	@echo "$(INFO_EMOJI) Staging changes..."
	@git add .
	@git commit -m "$(m)"
	@git push $(GIT_REMOTE) $(GIT_BRANCH)
	@echo "$(CHECK_EMOJI) Changes pushed to $(GIT_BRANCH)!"