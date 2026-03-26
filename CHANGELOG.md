# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial release of Anker Solix Exporter
- Continuous data export from Anker Solix API to InfluxDB
- Resume functionality to avoid duplicate imports
- Rate limiting awareness with configurable poll intervals
- Kubernetes deployment via Helm chart
- Multi-architecture Docker images (amd64, arm64, armv7)
- Comprehensive documentation and examples
- Docker Compose setup for local development
- GitHub Actions CI/CD pipeline
- Structured logging with configurable levels
- Configuration via YAML files and environment variables
- Graceful shutdown handling
- Persistent state management for resume functionality

### Security
- Non-root container user
- Read-only root filesystem
- Kubernetes security contexts
- Secret management for credentials

## [0.1.0] - 2026-03-04

### Added
- Initial project structure
- Core exporter functionality
- Anker Solix API client
- InfluxDB v2 writer
- Resume state management
- Configuration system
- Helm chart for Kubernetes
- Dockerfile and Docker Compose
- Documentation
- Unit tests
- Build automation (Makefile)
- GitHub Actions workflows

[Unreleased]: https://github.com/yourusername/anker-solix-exporter/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/yourusername/anker-solix-exporter/releases/tag/v0.1.0
