# Contributing to applecontainer-go

First off, thank you for taking the time to contribute! 🎉

We welcome and appreciate all contributions, whether it's reporting bugs, suggesting new features, improving documentation, or writing code.

The following guidelines are designed to help you get started quickly and make the contribution process smooth and productive for everyone.

---

## Code of Conduct

By participating in this project, you agree to maintain a respectful, welcoming, and collaborative environment. Please be kind and helpful to all participants.

---

## How Can I Contribute?

### 1. Reporting Bugs
* Search existing issues to see if the bug has already been reported.
* If not, open a new issue with a clear title and description.
* Include:
  * Your macOS version.
  * Your Go version (`go version`).
  * The version of the `container` CLI.
  * Steps to reproduce the issue, including a minimal code snippet if applicable.
  * Expected vs. actual behavior.

### 2. Suggesting Features
* Open an issue describing the feature you would like to see.
* Explain the use case and why this feature would be valuable to users of `applecontainer-go`.

### 3. Submitting Pull Requests (PRs)
* Fork the repository and create your branch from `main`.
* Write clean, documented, and idiomatic Go code.
* Ensure all existing tests pass and add new tests for your changes.
* Keep your PR focused on a single change or feature.

---

## Local Development Setup

To develop and test `applecontainer-go` locally, ensure you have the following prerequisites installed on your development machine.

### Prerequisites
* **Operating System**: macOS 26+ (Apple native container orchestration environment).
* **Architecture**: Apple Silicon (M1, M2, M3, M4, or newer).
* **Go**: Go 1.26 or higher (configured with Go Modules).
* **Toolchain**: GNU `make` for running task shortcuts.
* **CLI Dependency**: The Apple native `container` CLI must be installed and active (e.g. `/opt/homebrew/bin/container` or `/usr/local/bin/container`).

### Setup Instructions
1. Clone your fork of the repository:
   ```bash
   git clone https://github.com/<your-username>/applecontainer-go.git
   cd applecontainer-go
   ```
2. Verify dependencies:
   ```bash
   go mod tidy
   go mod verify
   ```

---

## Makefile Targets & Verification

The project includes a `Makefile` to simplify common tasks. Please use these targets to check your work before pushing.

### 1. Code Formatting
Make sure your code is formatted correctly using the standard Go formatting tool:
```bash
go fmt ./...
```

### 2. Running Unit Tests
Unit tests use mocking and fakes, which means they do not require a live Apple container runtime environment. You can run them anywhere:
```bash
make test
```

### 3. Running Code Coverage
To run the tests and generate a coverage report locally:
```bash
make test-coverage
```
This writes the coverage profile to `coverage.out`, which is ignored by git.

### 4. Running Integration Tests
Integration tests run real container scenarios. These require a macOS machine with a running `container` CLI daemon:
```bash
APPLECONTAINER_INTEGRATION=1 go test -tags integration -v ./examples/...
```

### 5. Code Quality & Linting
Run `golangci-lint` to check for code quality and style violations:
```bash
make lint
```
*(Note: Requires `golangci-lint` to be installed on your machine. The CI will also run this lint check on every PR).*

### 6. Security & Vulnerability Scanning
Run security scanners to identify potential vulnerabilities:
```bash
# Run gosec to find security hotspots
make sec

# Run govulncheck to find known vulnerabilities in dependencies
make vuln-check
```

---

## Git Commit Guidelines

To keep the history clean and readable, please follow these commit conventions:
* Use the imperative mood in commit summaries (e.g., "Add HTTP wait strategy" instead of "Added HTTP wait strategy").
* Keep the first line (summary) under 50 characters.
* If necessary, provide a blank line followed by a more detailed description of the changes.
* Reference issues or PRs if applicable.

---

## Pull Request Guidelines

Before submitting your pull request, double-check that you have:
1. Formatted your code with `go fmt ./...`.
2. Verified all unit tests pass locally with `make test`.
3. Verified the lint and security checks pass with `make lint`, `make sec`, and `make vuln-check`.
4. Added unit (and/or integration) tests covering your new code.
5. Documented any new options or features in the codebase and public APIs.

Once your PR is submitted, GitHub Actions will automatically run the verification suite (lint, vulnerability scan, security scan, and unit tests). A project maintainer will review your work and provide feedback.
