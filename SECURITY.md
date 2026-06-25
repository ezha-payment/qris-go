# Security Policy

## Supported versions

This library is pre-1.0. Security fixes are applied to the latest released
minor version only.

## Reporting a vulnerability

Please do not open a public issue for security vulnerabilities.

Report privately via GitHub's [private vulnerability reporting](https://github.com/ezha-payment/qris-go/security/advisories/new),
or by email to the maintainer. Include:

- a description of the issue and its impact,
- steps to reproduce or a proof of concept,
- affected version(s).

You can expect an acknowledgement within a few business days. Once the issue
is confirmed and a fix is prepared, a release and advisory will be published.

## Scope

This library parses and generates QRIS payloads. It does not perform payment
authorization, settlement, or network communication. It implements public
specifications and is not a substitute for official certification. Validate
all payloads against the authoritative ASPI/EMVCo specifications before use in
production.
