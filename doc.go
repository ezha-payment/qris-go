// Package qris provides parsing, generation, validation, and conversion
// of QRIS payment payloads.
//
// QRIS (Quick Response Code Indonesian Standard) is Indonesia's national
// QR payment standard, compliant with the EMVCo Merchant Presented Mode
// (MPM) specification. Payloads are ASCII TLV-encoded (Tag-Length-Value)
// strings terminated by a CRC-16/CCITT-FALSE checksum.
//
// # Features
//
//   - Parser with CRC verification for EMVCo MPM TLV payloads.
//   - Merchant account info parsing for tags 26-51 (PSP entries and the
//     national QRIS routing identity in tag 51).
//   - Additional data (tag 62) parsing.
//   - Builder for generating payloads with automatic length and CRC
//     computation.
//   - Strict validation aligned with the ASPI/EMVCo specification.
//   - ConvertStaticToDynamic for the acquirer point-of-sale use case.
//   - Typed constants for merchant criteria, currency, country, and GUI.
//
// The library depends only on the Go standard library.
//
// # Parsing
//
//	p, err := qris.Parse(payload)
//	if err != nil {
//		// payload was malformed or its CRC did not verify
//	}
//	fmt.Println(p.MerchantName, p.MPAN())
//
// The parser is intentionally permissive so it can decode real-world
// payloads. Use [Validate] to enforce the stricter rules required to
// emit a conformant payload.
//
// # Building
//
//	payload, err := qris.NewBuilder().
//		Static().
//		MerchantName("WARUNG SAMPLE").
//		MerchantCity("JAKARTA").
//		MCC("4812").
//		AddQRISAccount("9360091500001234567", "ID1020012345678", string(qris.CriteriaUMI)).
//		Build()
//
// [Builder.Build] validates the result and recomputes every TLV length
// prefix and the trailing CRC, so callers never set those manually.
//
// # Converting static to dynamic
//
//	dynamic, err := qris.ConvertStaticToDynamic(staticPayload, "25000",
//		qris.WithReferenceLabel("INV-2024-001"),
//	)
//
// ConvertStaticToDynamic preserves every original field and only changes
// the point of initiation method, the amount, and any requested
// additional-data fields.
//
// # Specification references
//
//   - ASPI QRIS Specification
//   - EMVCo Merchant Presented Mode (MPM) Specification v1.1
//   - Bank Indonesia PADG No. 21/18/PADG/2019
//
// This library implements public specifications. It is not affiliated
// with Bank Indonesia or ASPI, and is not a substitute for official
// certification.
package qris
