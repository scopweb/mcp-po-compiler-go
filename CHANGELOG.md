# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.1] - 2026-02-07

### Changed

- Updated Go version from 1.21 to 1.25.0 (toolchain go1.25.7)
- Updated golang.org/x/text from v0.3.8 to v0.33.0

### Security

- Fixed all vulnerabilities detected by govulncheck
- Passed govulncheck with 0 vulnerabilities

## [1.0.0] - Initial Release

### Added

- MCP PO compiler server
- PO file compilation to MO format
- Test files for PO compiler validation
