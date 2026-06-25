package qris

import (
	"fmt"
	"strings"
	"testing"
)

// --- Test helpers ---

// tlv constructs a TLV-encoded string with auto-computed length.
// This eliminates a class of bugs from manual length counting.
func tlv(tag, value string) string {
	return fmt.Sprintf("%s%02d%s", tag, len(value), value)
}

// buildPayload assembles parts and appends "6304" + correct CRC.
func buildPayload(parts ...string) string {
	dataNoCRC := strings.Join(parts, "") + "6304"
	crc := formatCRC(computeCRC([]byte(dataNoCRC)))
	return dataNoCRC + crc
}

// buildMerchantAccount constructs a nested TLV value for a tag 26-51
// merchant account block. Empty fields are omitted (not all PSPs use
// all sub-tags).
func buildMerchantAccount(gui, mpan, mid, criteria string) string {
	var b strings.Builder
	if gui != "" {
		b.WriteString(tlv("00", gui))
	}
	if mpan != "" {
		b.WriteString(tlv("01", mpan))
	}
	if mid != "" {
		b.WriteString(tlv("02", mid))
	}
	if criteria != "" {
		b.WriteString(tlv("03", criteria))
	}
	return b.String()
}

// --- Tests ---

func TestParse_Tag51Only_NationalQRIS(t *testing.T) {
	merchantAcc := buildMerchantAccount(
		"ID.CO.QRIS.WWW",
		"9360091500001234567",
		"ID1020012345678",
		"UMI",
	)
	payload := buildPayload(
		tlv("00", "01"),
		tlv("01", "11"),
		tlv("51", merchantAcc),
		tlv("52", "4812"),
		tlv("53", "360"),
		tlv("58", "ID"),
		tlv("59", "WARUNG SAMPLE"),
		tlv("60", "JAKARTA"),
	)

	result, err := Parse(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v\npayload: %s", err, payload)
	}

	acc, ok := result.QRISMerchantAccount()
	if !ok {
		t.Fatal("expected QRIS merchant account (tag 51), got none")
	}
	checks := []struct{ name, got, want string }{
		{"GUI", acc.GloballyUniqueIdentifier, "ID.CO.QRIS.WWW"},
		{"MPAN", acc.MPAN, "9360091500001234567"},
		{"MID", acc.MID, "ID1020012345678"},
		{"MerchantCriteria", acc.MerchantCriteria, "UMI"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.name, c.got, c.want)
		}
	}

	if result.MPAN() != "9360091500001234567" {
		t.Errorf("Payload.MPAN() = %q, want %q",
			result.MPAN(), "9360091500001234567")
	}
	if result.MID() != "ID1020012345678" {
		t.Errorf("Payload.MID() = %q, want %q",
			result.MID(), "ID1020012345678")
	}
	if len(result.PSPMerchantAccounts()) != 0 {
		t.Errorf("expected 0 PSP entries, got %d",
			len(result.PSPMerchantAccounts()))
	}
}

func TestParse_Tag26Only_PSPDirect(t *testing.T) {
	merchantAcc := buildMerchantAccount(
		"ID.DANA.WWW",
		"9360091512345678900",
		"1234567890",
		"",
	)
	payload := buildPayload(
		tlv("00", "01"),
		tlv("01", "11"),
		tlv("26", merchantAcc),
		tlv("52", "5999"),
		tlv("53", "360"),
		tlv("58", "ID"),
		tlv("59", "DANA MERCHANT"),
		tlv("60", "JAKARTA"),
	)

	result, err := Parse(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	psps := result.PSPMerchantAccounts()
	if len(psps) != 1 {
		t.Fatalf("expected 1 PSP entry, got %d", len(psps))
	}
	if psps[0].GloballyUniqueIdentifier != "ID.DANA.WWW" {
		t.Errorf("GUI = %q, want %q",
			psps[0].GloballyUniqueIdentifier, "ID.DANA.WWW")
	}
	if psps[0].MPAN != "9360091512345678900" {
		t.Errorf("MPAN = %q, want %q",
			psps[0].MPAN, "9360091512345678900")
	}

	if _, ok := result.QRISMerchantAccount(); ok {
		t.Error("expected no tag 51, got one")
	}

	// MPAN()/MID() should fall back to PSP entry
	if result.MPAN() != "9360091512345678900" {
		t.Errorf("MPAN() fallback = %q, want PSP MPAN", result.MPAN())
	}
}

func TestParse_Tag26And51_Both(t *testing.T) {
	pspAcc := buildMerchantAccount(
		"ID.DANA.WWW",
		"9360091599999999999",
		"1111111111",
		"",
	)
	qrisAcc := buildMerchantAccount(
		"ID.CO.QRIS.WWW",
		"9360091500001234567",
		"ID1020012345678",
		"UMI",
	)

	payload := buildPayload(
		tlv("00", "01"),
		tlv("01", "11"),
		tlv("26", pspAcc),
		tlv("51", qrisAcc),
		tlv("52", "4812"),
		tlv("53", "360"),
		tlv("58", "ID"),
		tlv("59", "MULTI MERCHANT"),
		tlv("60", "JAKARTA"),
	)

	result, err := Parse(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.MerchantAccountInfo) != 2 {
		t.Fatalf("expected 2 merchant account entries, got %d",
			len(result.MerchantAccountInfo))
	}

	// Tag 51 takes precedence in helpers
	if result.MPAN() != "9360091500001234567" {
		t.Errorf("MPAN() = %q, want tag-51 MPAN", result.MPAN())
	}
}

func TestParse_AdditionalData(t *testing.T) {
	additional := tlv("01", "INV") +
		tlv("03", "STORE01") +
		tlv("07", "T01")

	qrisAcc := buildMerchantAccount(
		"ID.CO.QRIS.WWW",
		"9360091500001234567",
		"ID1020012345678",
		"UMI",
	)

	payload := buildPayload(
		tlv("00", "01"),
		tlv("01", "12"),              // Dynamic
		tlv("51", qrisAcc),
		tlv("52", "4812"),
		tlv("53", "360"),
		tlv("54", "100000"),           // Transaction Amount
		tlv("58", "ID"),
		tlv("59", "DYNAMIC MERCH"),
		tlv("60", "JAKARTA"),
		tlv("62", additional),
	)

	result, err := Parse(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.AdditionalData == nil {
		t.Fatal("expected AdditionalData, got nil")
	}
	checks := []struct{ name, got, want string }{
		{"BillNumber", result.AdditionalData.BillNumber, "INV"},
		{"StoreLabel", result.AdditionalData.StoreLabel, "STORE01"},
		{"TerminalLabel", result.AdditionalData.TerminalLabel, "T01"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.name, c.got, c.want)
		}
	}

	if !result.IsDynamic() {
		t.Error("expected IsDynamic() to be true")
	}
	if result.TransactionAmount != "100000" {
		t.Errorf("TransactionAmount = %q, want %q",
			result.TransactionAmount, "100000")
	}
}
