# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- QRIS payload parser with CRC-16/CCITT-FALSE validation
- Support for tags 26-51 (PSP and national QRIS routing)
- Tag 62 additional data parsing
- Builder API for generating QRIS payloads
- Validation layer aligned with ASPI/EMVCo spec
- ConvertStaticToDynamic for acquirer use case
- Typed constants for criteria, currency, country, GUI
