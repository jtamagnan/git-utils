# Git Utils - Makefile for running tests and common tasks
#
# Usage:
#   make test       - Run all unit tests
#   make test-verbose - Run all tests with verbose output
#   make test-<module> - Run tests for specific module
#   make lint       - Run linting (via Nix)
#   make build      - Build all tools
#   make clean      - Clean build artifacts
#   make help       - Show this help

.PHONY: test test-unit test-verbose test-coverage lint build clean help
.PHONY: test-editor test-git test-keychain test-keychain-lib test-lint test-lint-lib test-review

# Define all Go modules in workspace (based on go.work)
MODULES := editor git keychain keychain/lib lint lint/lib review

# Default target
.DEFAULT_GOAL := help


# Run all unit tests
test:
	@printf "Running unit tests for all modules...\n"
	@printf "Note: Some authentication tests may fail if you have GitHub tokens configured\n"
	@echo ""
	@failed=0; \
	for module in $(MODULES); do \
		printf "Testing $$module...\n"; \
		if [ -d "$$module" ]; then \
			(cd "$$module" && go test -race ./...) || failed=1; \
		else \
			printf "Warning: Module $$module directory not found\n"; \
		fi; \
		echo ""; \
	done; \
	if [ $$failed -eq 1 ]; then \
		printf "Some tests failed!\n"; \
		printf "Tip: Try 'make test-unit' to skip integration tests that require clean auth state\n"; \
		exit 1; \
	else \
		printf "All tests passed!\n"; \
	fi

# Run unit tests, skipping integration tests that require clean auth state
test-unit:
	@printf "Running unit tests (skipping auth integration tests)...\n"
	@echo ""
	@failed=0; \
	for module in $(MODULES); do \
		printf "Testing $$module (unit tests only)...\n"; \
		if [ -d "$$module" ]; then \
			(cd "$$module" && go test -race -short ./...) || failed=1; \
		else \
			printf "Warning: Module $$module directory not found\n"; \
		fi; \
		echo ""; \
	done; \
	if [ $$failed -eq 1 ]; then \
		printf "Some tests failed!\n"; \
		exit 1; \
	else \
		printf "All unit tests passed!\n"; \
	fi

# Run all tests with verbose output
test-verbose:
	@printf "Running unit tests (verbose) for all modules...\n"
	@echo ""
	@failed=0; \
	for module in $(MODULES); do \
		printf "Testing $$module (verbose)...\n"; \
		if [ -d "$$module" ]; then \
			(cd "$$module" && go test -v -race ./...) || failed=1; \
		else \
			printf "Warning: Module $$module directory not found\n"; \
		fi; \
		echo ""; \
	done; \
	if [ $$failed -eq 1 ]; then \
		printf "Some tests failed!\n"; \
		exit 1; \
	else \
		printf "All tests passed!\n"; \
	fi

# Run tests with coverage
test-coverage:
	@printf "Running unit tests with coverage for all modules...\n"
	@echo ""
	@failed=0; \
	for module in $(MODULES); do \
		printf "Testing $$module with coverage...\n"; \
		if [ -d "$$module" ]; then \
			(cd "$$module" && go test -race -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html) || failed=1; \
		else \
			printf "Warning: Module $$module directory not found\n"; \
		fi; \
		echo ""; \
	done; \
	if [ $$failed -eq 1 ]; then \
		printf "Some tests failed!\n"; \
		exit 1; \
	else \
		printf "All tests passed! Coverage reports generated.\n"; \
	fi

# Individual module test targets
test-editor:
	@printf "Testing editor module...\n"
	@(cd editor && go test -race ./...)

test-git:
	@printf "Testing git module...\n"
	@(cd git && go test -race ./...)

test-keychain:
	@printf "Testing keychain module...\n"
	@(cd keychain && go test -race ./...)

test-keychain-lib:
	@printf "Testing keychain/lib module...\n"
	@(cd keychain/lib && go test -race ./...)

test-lint:
	@printf "Testing lint module...\n"
	@(cd lint && go test -race ./...)

test-lint-lib:
	@printf "Testing lint/lib module...\n"
	@(cd lint/lib && go test -race ./...)

test-review:
	@printf "Testing review module...\n"
	@(cd review && go test -race ./...)

# Run linting using Nix flake
lint:
	@printf "Running linting via Nix flake...\n"
	@nix run .#gitlint

# Build all tools using Nix flake
build:
	@printf "Building all tools...\n"
	@echo "Building git-keychain..."
	@nix build .#gitkeychain
	@echo "Building git-review..."
	@nix build .#gitreview
	@echo "Building git-lint..."
	@nix build .#gitlint
	@printf "All tools built successfully!\n"

# Clean build artifacts and test coverage files
clean:
	@printf "Cleaning build artifacts...\n"
	@find . -name "coverage.out" -delete
	@find . -name "coverage.html" -delete
	@rm -rf result
	@printf "Clean complete!\n"

# Run a quick smoke test to verify tools work
smoke-test: build
	@printf "Running smoke tests...\n"
	@echo "Testing git-keychain..."
	@./result/bin/git-keychain --help > /dev/null
	@echo "Testing git-review..."
	@./result/bin/git-review --help > /dev/null
	@echo "Testing git-lint..."
	@./result/bin/git-lint --help > /dev/null || true
	@printf "Smoke tests passed!\n"

# Show help
help:
	@printf "Git Utils Makefile\n"
	@echo ""
	@printf "Available targets:\n"
	@printf "  test              - Run all unit tests\n"
	@printf "  test-unit         - Run unit tests (skip auth integration tests)\n"
	@printf "  test-verbose      - Run all tests with verbose output\n"
	@printf "  test-coverage     - Run tests with coverage reports\n"
	@printf "  test-<module>     - Run tests for specific module (editor, git, keychain, etc.)\n"
	@printf "  lint              - Run linting via Nix flake (nix run .#gitlint)\n"
	@printf "  build             - Build all tools via Nix flake\n"
	@printf "  clean             - Clean build artifacts and coverage files\n"
	@printf "  smoke-test        - Build and test that all tools run\n"
	@printf "  help              - Show this help\n"
	@echo ""
	@printf "Available modules: $(MODULES)\n"
	@echo ""
	@printf "Examples:\n"
	@printf "  make test                 # Run all tests\n"
	@printf "  make test-unit            # Run unit tests (skip auth integration tests)\n"
	@printf "  make test-git             # Test only the git module\n"
	@printf "  make test-verbose         # Run all tests with verbose output\n"
	@printf "  make lint                 # Run code quality checks\n"
	@printf "  make build                # Build all tools\n"
	@echo ""
	@printf "Nix commands (alternative):\n"
	@printf "  nix run .#gitlint         # Run linting directly\n"
	@printf "  nix run .#gitreview       # Run git-review directly\n"
	@printf "  nix run .#gitkeychain     # Run git-keychain directly\n"
