package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	qris "github.com/ezha-payment/qris-go"
)

func main() {
	var payload string

	if len(os.Args) >= 2 {
		// Use payload from command-line argument.
		payload = os.Args[1]
	} else {
		// Fallback: build a sample payload programmatically.
		payload = buildSamplePayload()
		fmt.Println("(No payload arg provided — using built-in sample)")
		fmt.Println()
	}

	result, err := qris.Parse(payload)
	if err != nil {
		log.Fatalf("Parse failed: %v\n", err)
	}

	printPayload(result)
}

func buildSamplePayload() string {
	qrisAccount := tlv("00", string(qris.GUIQRISNational)) +
		tlv("01", "9360091500001234567") +
		tlv("02", "ID1020012345678") +
		tlv("03", string(qris.CriteriaUMI))

	parts := []string{
		tlv("00", "01"),
		tlv("01", "11"),
		tlv("51", qrisAccount),
		tlv("52", "4812"),
		tlv("53", string(qris.CurrencyIDR)),
		tlv("58", string(qris.CountryID)),
		tlv("59", "WARUNG SAMPLE"),
		tlv("60", "JAKARTA"),
	}
	dataNoCRC := strings.Join(parts, "") + "6304"
	return dataNoCRC + mustCRC(dataNoCRC)
}

func printPayload(result *qris.Payload) {
	fmt.Println("=== QRIS Payload ===")
	fmt.Printf("Format Indicator     : %s\n", result.PayloadFormatIndicator)
	fmt.Printf("Initiation           : %s\n", initiationLabel(result))
	fmt.Printf("Merchant Name        : %s\n", result.MerchantName)
	fmt.Printf("Merchant City        : %s\n", result.MerchantCity)
	fmt.Printf("Postal Code          : %s\n", result.PostalCode)
	fmt.Printf("MCC                  : %s\n", result.MerchantCategoryCode)
	fmt.Printf("Currency             : %s (%s)\n",
		result.TransactionCurrency, qris.CurrencyCode(result.TransactionCurrency).Description())
	fmt.Printf("Transaction Amount   : %s\n", emptyAsDash(result.TransactionAmount))
	fmt.Printf("Country              : %s (%s)\n",
		result.CountryCode, qris.CountryCode(result.CountryCode).Description())
	fmt.Printf("CRC                  : %s\n", result.CRC)

	fmt.Println("\n=== Merchant Account Info ===")
	if acc, ok := result.QRISMerchantAccount(); ok {
		fmt.Println("[Tag 51 — National QRIS Routing]")
		printAccount(acc)
	}
	for tag, acc := range result.MerchantAccountInfo {
		if tag == "51" {
			continue
		}
		fmt.Printf("[Tag %s — PSP Entry]\n", tag)
		printAccount(acc)
	}

	if result.AdditionalData != nil {
		fmt.Println("\n=== Additional Data (Tag 62) ===")
		printAdditional(result.AdditionalData)
	}

	fmt.Println("\n=== Quick Access (Helper Methods) ===")
	fmt.Printf("MPAN                 : %s\n", emptyAsDash(result.MPAN()))
	fmt.Printf("MID                  : %s\n", emptyAsDash(result.MID()))
}

func printAccount(acc qris.MerchantAccount) {
	fmt.Printf("  GUI                : %s (%s)\n",
		acc.GloballyUniqueIdentifier, qris.GUI(acc.GloballyUniqueIdentifier).Description())
	fmt.Printf("  MPAN               : %s\n", emptyAsDash(acc.MPAN))
	fmt.Printf("  MID                : %s\n", emptyAsDash(acc.MID))
	if acc.MerchantCriteria != "" {
		fmt.Printf("  Merchant Criteria  : %s (%s)\n",
			acc.MerchantCriteria, qris.MerchantCriteria(acc.MerchantCriteria).Description())
	} else {
		fmt.Printf("  Merchant Criteria  : —\n")
	}
	if len(acc.Raw) > 0 {
		fmt.Printf("  Raw sub-tags       : %v\n", acc.Raw)
	}
}

func printAdditional(ad *qris.AdditionalData) {
	fields := []struct{ name, value string }{
		{"Bill Number", ad.BillNumber},
		{"Mobile Number", ad.MobileNumber},
		{"Store Label", ad.StoreLabel},
		{"Loyalty Number", ad.LoyaltyNumber},
		{"Reference Label", ad.ReferenceLabel},
		{"Customer Label", ad.CustomerLabel},
		{"Terminal Label", ad.TerminalLabel},
		{"Purpose", ad.PurposeOfTransaction},
	}
	for _, f := range fields {
		if f.value != "" {
			fmt.Printf("  %-18s : %s\n", f.name, f.value)
		}
	}
	if len(ad.Raw) > 0 {
		fmt.Printf("  Raw sub-tags       : %v\n", ad.Raw)
	}
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

func emptyAsDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

// tlv constructs a TLV-encoded string with auto-computed length.
func tlv(tag, value string) string {
	return fmt.Sprintf("%s%02d%s", tag, len(value), value)
}

// mustCRC computes CRC-16/CCITT-FALSE for the sample payload builder.
func mustCRC(data string) string {
	crc := uint16(0xFFFF)
	for _, b := range []byte(data) {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return fmt.Sprintf("%04X", crc)
}
