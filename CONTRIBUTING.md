# Contributing to GomPDF

Thank you for your interest in contributing to GomPDF! This document provides guidelines and instructions for contributing to this project.

## Code of Conduct

Please read and follow our [Code of Conduct](CODE_OF_CONDUCT.md).

## Getting Started

1. Fork the repository on GitHub
2. Clone your fork: `git clone https://github.com/YOUR-USERNAME/gompdf.git`
3. Add the upstream repository: `git remote add upstream https://github.com/henrrius/gompdf.git`
4. Create a branch for your work: `git checkout -b feature/your-feature-name`

## Development Setup

1. Ensure you have Go 1.18 or later installed
2. Install development dependencies: `make dev-deps`
3. Run tests to verify your setup: `make test`

## Development Workflow

1. Make your changes in your feature branch
2. Add tests for your changes
3. Run tests: `make test`
4. Run linter: `make lint`
5. Format your code: `make fmt`
6. Commit your changes with a descriptive commit message
7. Push your branch to your fork: `git push origin feature/your-feature-name`
8. Create a pull request from your fork to the main repository

## Pull Request Process

1. Ensure your PR includes tests for any new functionality
2. Update documentation as needed
3. Make sure all tests pass and linting issues are resolved
4. Sign your commits with the Developer Certificate of Origin (DCO)
5. Wait for a maintainer to review your PR
6. Address any feedback from reviewers

## Commit Messages

Please follow these guidelines for commit messages:

- Use the present tense ("Add feature" not "Added feature")
- Use the imperative mood ("Move cursor to..." not "Moves cursor to...")
- Limit the first line to 72 characters or less
- Reference issues and pull requests after the first line

## Developer Certificate of Origin (DCO)

All contributions must be signed off with the DCO. This certifies that you have the right to contribute the code you are submitting.

To sign your commits, add the `-s` flag to your git commit command:

```
git commit -s -m "Your commit message"
```

## Testing

- Write tests for all new functionality
- Ensure existing tests pass with your changes
- Aim for high test coverage

## Documentation

- Update documentation for any changed functionality
- Document new features thoroughly
- Use clear, concise language

## Reporting Bugs

- Use the GitHub issue tracker
- Include detailed steps to reproduce the bug
- Include information about your environment (OS, Go version, etc.)
- Include any relevant logs or error messages

## Feature Requests

- Use the GitHub issue tracker
- Clearly describe the feature and its use case
- Discuss potential implementations if possible

## Questions

If you have questions about contributing, please:

1. Check the documentation
2. Search existing issues and discussions
3. Open a new issue with the "question" template if needed

Thank you for contributing to GomPDF!
