# Changelog

All notable changes to Gardener will be documented in this file.

This project follows the spirit of [Keep a Changelog](https://keepachangelog.com/) and uses semantic versioning once tagged releases begin.

## [Unreleased]

### Added

- Task dashboard runtime diagnostics, idle-output watchdog cues, and mock runner smoke-test mode.
- Initial public repository structure.
- Multi-platform packaging scripts for Windows and macOS.
- Local-first Gardener web server and static UI.
- Multi-instance relay deployment helper and installer examples.

### Changed

- Removed the automatic stage cap; Gardener continues into new stages until completion, user stop, or CLI/model failure.

### Security

- Public-release safety checklist for keeping local deployment secrets out of git.
