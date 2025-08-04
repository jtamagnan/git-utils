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

# TODO(jat) remove color

# Run all unit tests
test:
	@printf "\033[0;32mRunning unit tests for all modules...\033[0m\n"
	@printf "\033[0;33mNote: Some authentication tests may fail if you have GitHub tokens configured\033[0m\n"
	@echo ""
	@failed=0; \
	for module in $(MODULES); do \
		printf "\033[0;33mTesting $$module...\033[0m\n"; \
		if [ -d "$$module" ]; then \
			(cd "$$module" && go test -race ./...) || failed=1; \
		else \
			printf "\033[0;31mWarning: Module $$module directory not found\033[0m\n"; \
		fi; \
		echo ""; \
	done; \
	if [ $$failed -eq 1 ]; then \
		printf "\033[0;31mSome tests failed!\033[0m\n"; \
		printf "\033[0;33mTip: Try 'make test-unit' to skip integration tests that require clean auth state\033[0m\n"; \
		exit 1; \
	else \
		printf "\033[0;32mAll tests passed!\033[0m\n"; \
	fi

# Run unit tests, skipping integration tests that require clean auth state
test-unit:
	@printf "\033[0;32mRunning unit tests (skipping auth integration tests)...\033[0m\n"
	@echo ""
	@failed=0; \
	for module in $(MODULES); do \
		printf "\033[0;33mTesting $$module (unit tests only)...\033[0m\n"; \
		if [ -d "$$module" ]; then \
			(cd "$$module" && go test -race -short ./...) || failed=1; \
		else \
			printf "\033[0;31mWarning: Module $$module directory not found\033[0m\n"; \
		fi; \
		echo ""; \
	done; \
	if [ $$failed -eq 1 ]; then \
		printf "\033[0;31mSome tests failed!\033[0m\n"; \
		exit 1; \
	else \
		printf "\033[0;32mAll unit tests passed!\033[0m\n"; \
	fi

# Run all tests with verbose output
test-verbose:
	@printf "\033[0;32mRunning unit tests (verbose) for all modules...\033[0m\n"
	@echo ""
	@failed=0; \
	for module in $(MODULES); do \
		printf "\033[0;33mTesting $$module (verbose)...\033[0m\n"; \
		if [ -d "$$module" ]; then \
			(cd "$$module" && go test -v -race ./...) || failed=1; \
		else \
			printf "\033[0;31mWarning: Module $$module directory not found\033[0m\n"; \
		fi; \
		echo ""; \
	done; \
	if [ $$failed -eq 1 ]; then \
		printf "\033[0;31mSome tests failed!\033[0m\n"; \
		exit 1; \
	else \
		printf "\033[0;32mAll tests passed!\033[0m\n"; \
	fi

# Run tests with coverage
test-coverage:
	@printf "\033[0;32mRunning unit tests with coverage for all modules...\033[0m\n"
	@echo ""
	@failed=0; \
	for module in $(MODULES); do \
		printf "\033[0;33mTesting $$module with coverage...\033[0m\n"; \
		if [ -d "$$module" ]; then \
			(cd "$$module" && go test -race -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html) || failed=1; \
		else \
			printf "\033[0;31mWarning: Module $$module directory not found\033[0m\n"; \
		fi; \
		echo ""; \
	done; \
	if [ $$failed -eq 1 ]; then \
		printf "\033[0;31mSome tests failed!\033[0m\n"; \
		exit 1; \
	else \
		printf "\033[0;32mAll tests passed! Coverage reports generated.\033[0m\n"; \
	fi

# Individual module test targets
test-editor:
	@printf "\033[0;32mTesting editor module...\033[0m\n"
	@(cd editor && go test -race ./...)

test-git:
	@printf "\033[0;32mTesting git module...\033[0m\n"
	@(cd git && go test -race ./...)

test-keychain:
	@printf "\033[0;32mTesting keychain module...\033[0m\n"
	@(cd keychain && go test -race ./...)

test-keychain-lib:
	@printf "\033[0;32mTesting keychain/lib module...\033[0m\n"
	@(cd keychain/lib && go test -race ./...)

test-lint:
	@printf "\033[0;32mTesting lint module...\033[0m\n"
	@(cd lint && go test -race ./...)

test-lint-lib:
	@printf "\033[0;32mTesting lint/lib module...\033[0m\n"
	@(cd lint/lib && go test -race ./...)

test-review:
	@printf "\033[0;32mTesting review module...\033[0m\n"
	@(cd review && go test -race ./...)

# Run linting using Nix flake
lint:
	@printf "\033[0;32mRunning linting via Nix flake...\033[0m\n"
	@nix run .#gitlint

# Build all tools using Nix flake
build:
	@printf "\033[0;32mBuilding all tools...\033[0m\n"
	@echo "Building git-keychain..."
	@nix build .#gitkeychain
	@echo "Building git-review..."
	@nix build .#gitreview
	@echo "Building git-lint..."
	@nix build .#gitlint
	@printf "\033[0;32mAll tools built successfully!\033[0m\n"

# Clean build artifacts and test coverage files
clean:
	@printf "\033[0;32mCleaning build artifacts...\033[0m\n"
	@find . -name "coverage.out" -delete
	@find . -name "coverage.html" -delete
	@rm -rf result
	@printf "\033[0;32mClean complete!\033[0m\n"

# Run a quick smoke test to verify tools work
smoke-test: build
	@printf "\033[0;32mRunning smoke tests...\033[0m\n"
	@echo "Testing git-keychain..."
	@./result/bin/git-keychain --help > /dev/null
	@echo "Testing git-review..."
	@./result/bin/git-review --help > /dev/null
	@echo "Testing git-lint..."
	@./result/bin/git-lint --help > /dev/null || true
	@printf "\033[0;32mSmoke tests passed!\033[0m\n"

# Show help
help:
	@printf "\033[0;32mGit Utils Makefile\033[0m\n"
	@echo ""
	@printf "\033[0;33mAvailable targets:\033[0m\n"
	@printf "  \033[0;32mtest\033[0m              - Run all unit tests\n"
	@printf "  \033[0;32mtest-unit\033[0m         - Run unit tests (skip auth integration tests)\n"
	@printf "  \033[0;32mtest-verbose\033[0m      - Run all tests with verbose output\n"
	@printf "  \033[0;32mtest-coverage\033[0m     - Run tests with coverage reports\n"
	@printf "  \033[0;32mtest-<module>\033[0m     - Run tests for specific module (editor, git, keychain, etc.)\n"
	@printf "  \033[0;32mlint\033[0m              - Run linting via Nix flake (nix run .#gitlint)\n"
	@printf "  \033[0;32mbuild\033[0m             - Build all tools via Nix flake\n"
	@printf "  \033[0;32mclean\033[0m             - Clean build artifacts and coverage files\n"
	@printf "  \033[0;32msmoke-test\033[0m        - Build and test that all tools run\n"
	@printf "  \033[0;32mhelp\033[0m              - Show this help\n"
	@echo ""
	@printf "\033[0;33mAvailable modules:\033[0m $(MODULES)\n"
	@echo ""
	@printf "\033[0;33mExamples:\033[0m\n"
	@printf "  make test                 # Run all tests\n"
	@printf "  make test-unit            # Run unit tests (skip auth integration tests)\n"
	@printf "  make test-git             # Test only the git module\n"
	@printf "  make test-verbose         # Run all tests with verbose output\n"
	@printf "  make lint                 # Run code quality checks\n"
	@printf "  make build                # Build all tools\n"
	@echo ""
	@printf "\033[0;33mNix commands (alternative):\033[0m\n"
	@printf "  nix run .#gitlint         # Run linting directly\n"
	@printf "  nix run .#gitreview       # Run git-review directly\n"
	@printf "  nix run .#gitkeychain     # Run git-keychain directly\n"
