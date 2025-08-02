# git-utils

This is my attempt at recreating some tools that I had written and
worked on at my previous companies and side projects. Although
inspiration has been taken from what I remember of those tools I am
writing these from scratch in golang as an opportunity to learn
golang.

## Tools

- **`review`** - Creates pull requests with automatic branch naming, PR templates, and commit message updates
- **`keychain`** - Manages GitHub tokens securely in macOS keychain
- **`lint`** - Code linting and formatting tools

## Configuration

The tools can be configured using git config. Available options:

### Review Tool

- **`review.openBrowser`** (boolean, default: `true`) - Whether to automatically open the pull request in your browser after creation
  ```bash
  # Disable automatic browser opening
  git config review.openBrowser false

  # Enable automatic browser opening (default)
  git config review.openBrowser true
  ```

- **`review.user-identifier`** (string) - User identifier for branch naming (falls back to `$USER` environment variable)
  ```bash
  # Set a custom user identifier for branch prefixes
  git config review.user-identifier "john"
  ```

### Usage Examples

```bash
# Create a PR with default settings (respects git config)
go run ./review

# Override git config and force browser opening
go run ./review --open-browser

# Create a draft PR with labels
go run ./review --draft --labels "feature,backend"
```
