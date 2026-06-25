# qris-go

A Go library for parsing, generating, validating, and converting QRIS payment payloads.

[![CI](https://github.com/ezha-payment/qris-go/actions/workflows/ci.yml/badge.svg)](https://github.com/ezha-payment/qris-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/ezha-payment/qris-go.svg)](https://pkg.go.dev/github.com/ezha-payment/qris-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
![Go Version](https://img.shields.io/badge/go-1.26%2B-blue.svg)
[![Release](https://img.shields.io/github/v/release/ezha-payment/qris-go.svg)](https://github.com/ezha-payment/qris-go/releases/latest)

## Why this exists

QRIS (Quick Response Code Indonesian Standard) is Indonesia's national QR payment standard. As of mid-2025 it serves around 57 million users and 39 million merchants, and it now supports cross-border payments across five countries, including Japan and South Korea, with further expansion across APEC economies underway. This library provides a dependency-free, spec-aligned toolkit for working with QRIS payloads in Go: decoding them, building them, validating them, and converting static QRs into dynamic ones at the point of sale.

## Features

- Parser for EMVCo Merchant Presented Mode TLV payloads with CRC-16/CCITT-FALSE verification
- Merchant account info parsing for tags 26-51 (PSP-specific entries and national QRIS routing)
- Additional data (tag 62) parsing
- Builder API for generating valid QRIS payloads with automatic length and CRC computation
- Validation layer aligned with the ASPI/EMVCo specification
- `ConvertStaticToDynamic` for the acquirer point-of-sale use case
- Typed constants for merchant criteria, currency, country, and globally unique identifiers
- Standard library only, no external dependencies

## Installation

```sh
go get github.com/ezha-payment/qris-go
```

## Quick start: parsing

```go
package main

import (
	"fmt"
	"log"

	qris "github.com/ezha-payment/qris-go"
)

func main() {
	p, err := qris.Parse(payload)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(p.MerchantName, p.MerchantCity)
	fmt.Println(p.MPAN(), p.MID())
	if p.IsStatic() {
		fmt.Println("static QR")
	}
}
```

## Usage: building a payload

The Builder computes every TLV length prefix and the trailing CRC automatically.
Typed constants are passed with an explicit string conversion.

```go
payload, err := qris.NewBuilder().
	Static().
	MerchantName("WARUNG SAMPLE").
	MerchantCity("JAKARTA").
	MCC("4812").
	Currency(string(qris.CurrencyIDR)).
	Country(string(qris.CountryID)).
	AddQRISAccount("9360091500001234567", "ID1020012345678", string(qris.CriteriaUMI)).
	Build()
if err != nil {
	log.Fatal(err)
}
fmt.Println(payload)
```

`Build` validates the payload before returning; an invalid configuration yields a
`*qris.ValidationError` listing every offending field.

## Usage: converting static to dynamic

A merchant displays a static QR; at checkout the point-of-sale embeds the
purchase amount and emits a dynamic payload. Every original field is preserved;
only the point of initiation method, amount, and optional additional data change.

```go
dynamic, err := qris.ConvertStaticToDynamic(staticPayload, "25000",
	qris.WithReferenceLabel("INV-2024-001"),
)
if err != nil {
	log.Fatal(err)
}
fmt.Println(dynamic)
```

Runnable versions of these examples live in [`examples/`](examples/).

## API overview

Full reference: [pkg.go.dev/github.com/ezha-payment/qris-go](https://pkg.go.dev/github.com/ezha-payment/qris-go).

| Symbol | Purpose |
| --- | --- |
| `Parse(string) (*Payload, error)` | Decode a QRIS payload, verifying its CRC. |
| `Payload` | Parsed payload with typed accessors (`MPAN`, `MID`, `IsStatic`, `IsDynamic`, `QRISMerchantAccount`, `PSPMerchantAccounts`). |
| `NewBuilder() *Builder` | Fluent builder for generating payloads. |
| `Validate(*Payload) error` | Strict ASPI/EMVCo validation, returning a `*ValidationError`. |
| `ConvertStaticToDynamic(payload, amount string, ...ConvertOption)` | Embed an amount into a static QR. |
| `WithReferenceLabel`, `WithTerminalLabel`, `WithStoreLabel` | Additional-data overrides for conversion. |
| `MerchantCriteria`, `CurrencyCode`, `CountryCode`, `GUI` | Typed constants for enum-like fields. |

## Specification references

- ASPI QRIS Specification
- EMVCo Merchant Presented Mode (MPM) Specification v1.1
- Bank Indonesia PADG No. 21/18/PADG/2019

## Disclaimer

This library implements public specifications. Not affiliated with Bank Indonesia
or ASPI. Not a substitute for official certification.

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for how to run
tests, code style, and the pull request process.

## License

Released under the [MIT License](LICENSE).
