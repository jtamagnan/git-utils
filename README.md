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

The tools support multiple configuration sources with the following precedence (highest to lowest):

1. **Command-line flags** (highest priority)
2. **Environment variables** (with `REVIEW_` prefix)
3. **User-level config files** (`~/.git-review.yaml` or `~/.config/git-review.yaml`)
4. **Hardcoded defaults** (lowest priority)

**Note**: For security and consistency, behavioral settings like `open-browser`, `draft`, and `labels` are intentionally **NOT** configurable per repository. These remain user-level preferences only.

### Review Tool Configuration

#### Environment Variables
All settings can be configured via environment variables with the `REVIEW_` prefix:

```bash
# Configure default behavior via environment variables
export REVIEW_OPEN_BROWSER=false
export REVIEW_DRAFT=true
export REVIEW_NO_VERIFY=false
export REVIEW_LABELS="auto-generated,needs-review"
```

#### User Config File
Create `~/.git-review.yaml` for persistent user preferences:

```yaml
# User-level preferences (applies to all repositories)
open-browser: false
draft: true
no-verify: false
labels:
  - "auto-generated"
  - "needs-review"
```

#### Available Settings

- **`open-browser`** (boolean, default: `true`) - Whether to automatically open the pull request in your browser after creation
- **`draft`** (boolean, default: `false`) - Whether to create pull requests as drafts by default
- **`no-verify`** (boolean, default: `false`) - Whether to skip pre-push checks by default
- **`labels`** (array/string, default: `[]`) - Default labels to add to pull requests

### Configuration Precedence Examples

```bash
# Set user preference to not open browser
echo "open-browser: false" > ~/.git-review.yaml

# Override with environment variable
REVIEW_OPEN_BROWSER=true go run ./review

# Override both with command-line flag
REVIEW_OPEN_BROWSER=true go run ./review --open-browser=false
```

### Usage Examples

```bash
# Use defaults from config files/environment
go run ./review

# Override specific settings
go run ./review --open-browser=false --draft --labels "hotfix,urgent"

# Use environment variables for this session
REVIEW_LABELS="feature,frontend" go run ./review --draft
```
