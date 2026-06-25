// Command convert turns a static QRIS payload into a dynamic one with
// an embedded amount, demonstrating qris.ConvertStaticToDynamic.
//
// Usage:
//
//	go run ./examples/convert <static-payload> <amount> [reference-label]
package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	qris "github.com/ezha-payment/qris-go"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: convert <static-payload> <amount> [reference-label]")
		os.Exit(2)
	}

	staticPayload := os.Args[1]
	amount := os.Args[2]

	var opts []qris.ConvertOption
	if len(os.Args) >= 4 && os.Args[3] != "" {
		opts = append(opts, qris.WithReferenceLabel(os.Args[3]))
	}

	dynamic, err := qris.ConvertStaticToDynamic(staticPayload, amount, opts...)
	if err != nil {
		if errors.Is(err, qris.ErrAlreadyDynamic) {
			log.Fatalf("input is already a dynamic QR: %v", err)
		}
		log.Fatalf("convert failed: %v", err)
	}

	// Parse the result back to show what changed.
	parsed, err := qris.Parse(dynamic)
	if err != nil {
		log.Fatalf("converted payload did not re-parse: %v", err)
	}

	fmt.Println("=== Converted to Dynamic ===")
	fmt.Printf("Initiation         : %s\n", initiationLabel(parsed))
	fmt.Printf("Transaction Amount : %s\n", parsed.TransactionAmount)
	fmt.Printf("Merchant Name      : %s\n", parsed.MerchantName)
	fmt.Printf("Merchant City      : %s\n", parsed.MerchantCity)
	if parsed.AdditionalData != nil && parsed.AdditionalData.ReferenceLabel != "" {
		fmt.Printf("Reference Label    : %s\n", parsed.AdditionalData.ReferenceLabel)
	}
	fmt.Printf("CRC                : %s\n", parsed.CRC)
	fmt.Println()
	fmt.Println("=== Payload ===")
	fmt.Println(dynamic)
}

func initiationLabel(p *qris.Payload) string {
	switch {
	case p.IsStatic():
		return "Static (11)"
	case p.IsDynamic():
		return "Dynamic (12)"
	default:
		return "Unknown"
	}
}
