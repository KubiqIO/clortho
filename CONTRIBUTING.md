# Contributing to Clortho

Thank you for your interest in contributing to Clortho! This document provides guidelines and instructions for contributing.

## Getting Started

### Prerequisites

- Go 1.25 or higher
- PostgreSQL database
- Docker (optional, for running tests)

### Development Setup

1. **Fork and clone the repository**
   ```bash
   git clone https://github.com/KubiqIO/clortho.git
   cd clortho
   ```

2. **Copy the example configuration**
   ```bash
   cp config.yaml.example config.yaml
   # Edit config.yaml with your database credentials
   ```

3. **Set up the database**
   ```bash
   make migrate-up
   ```

4. **Run tests to verify setup**
   ```bash
   make test
   ```

## How to Contribute

### Reporting Bugs

- Check existing [issues](https://github.com/KubiqIO/clortho/issues) first
- Use the bug report template if available
- Include steps to reproduce, expected vs actual behavior
- Include Go version, OS, and relevant configuration

### Suggesting Features

- Open an issue to discuss before implementing
- Explain the use case and why existing functionality doesn't suffice

### Submitting Pull Requests

1. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes**
   - Follow existing code style
   - Add tests for new functionality
   - Update documentation as needed

3. **Run checks before submitting**
   ```bash
   make test          # Run unit tests
   make test-e2e      # Run end-to-end tests (requires containers)
   make check-vuln    # Run vulnerability scan
   ```

4. **Commit with clear messages**
   ```
   feat: add support for custom license charsets
   fix: correct expiration calculation for monthly licenses
   docs: update API documentation for releases endpoint
   ```

5. **Push and open a Pull Request**
   - Reference any related issues
   - Describe what changed and why
   - Include any breaking changes

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Use meaningful variable and function names
- Add comments for non-obvious logic
- Keep functions focused and reasonably sized

## Testing

- Write tests for new functionality
- Ensure existing tests pass
- Use table-driven tests where appropriate
- Integration tests use testcontainers for PostgreSQL

## Questions?

Feel free to open an issue for questions or reach out to the maintainers.
