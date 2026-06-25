package qris

import (
	"strings"
	"testing"
)

// --- Test helpers ---

// assertAccountEqual compares a parsed MerchantAccount against the
// modeled sub-tag values it was built from. Raw is not compared here
// (the round-trip tests do not exercise Raw on accounts).
func assertAccountEqual(t *testing.T, label string, got MerchantAccount, gui, mpan, mid, criteria string) {
	t.Helper()
	if got.GloballyUniqueIdentifier != gui {
		t.Errorf("%s GUI: got %q, want %q", label, got.GloballyUniqueIdentifier, gui)
	}
	if got.MPAN != mpan {
		t.Errorf("%s MPAN: got %q, want %q", label, got.MPAN, mpan)
	}
	if got.MID != mid {
		t.Errorf("%s MID: got %q, want %q", label, got.MID, mid)
	}
	if got.MerchantCriteria != criteria {
		t.Errorf("%s criteria: got %q, want %q", label, got.MerchantCriteria, criteria)
	}
}

// buildAndParse builds the payload, fails the test on a build error,
// then parses it back, failing the test on a parse error. This is the
// core of every round-trip test.
func buildAndParse(t *testing.T, b *Builder) *Payload {
	t.Helper()
	payload, err := b.Build()
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}
	parsed, err := Parse(payload)
	if err != nil {
		t.Fatalf("Parse(Build()) failed: %v\npayload: %s", err, payload)
	}
	return parsed
}

// --- Round-trip tests ---

func TestBuild_RoundTrip_Static(t *testing.T) {
	b := NewBuilder().
		Static().
		MerchantName("WARUNG SAMPLE").
		MerchantCity("JAKARTA").
		PostalCode("12345").
		MCC("4812").
		AddQRISAccount("9360091500001234567", "ID1020012345678", "UMI")

	p := buildAndParse(t, b)

	if !p.IsStatic() {
		t.Errorf("expected static QR, got POIM %q", p.PointOfInitiationMethod)
	}
	if p.PayloadFormatIndicator != "01" {
		t.Errorf("PayloadFormatIndicator: got %q, want %q", p.PayloadFormatIndicator, "01")
	}
	if p.TransactionAmount != "" {
		t.Errorf("static QR should have no amount, got %q", p.TransactionAmount)
	}
	if p.MerchantName != "WARUNG SAMPLE" {
		t.Errorf("MerchantName: got %q", p.MerchantName)
	}
	if p.MerchantCity != "JAKARTA" {
		t.Errorf("MerchantCity: got %q", p.MerchantCity)
	}
	if p.PostalCode != "12345" {
		t.Errorf("PostalCode: got %q", p.PostalCode)
	}
	if p.MerchantCategoryCode != "4812" {
		t.Errorf("MCC: got %q", p.MerchantCategoryCode)
	}
	if p.TransactionCurrency != "360" {
		t.Errorf("Currency default: got %q, want %q", p.TransactionCurrency, "360")
	}
	if p.CountryCode != "ID" {
		t.Errorf("Country default: got %q, want %q", p.CountryCode, "ID")
	}

	acc, ok := p.QRISMerchantAccount()
	if !ok {
		t.Fatal("expected tag 51 national QRIS account")
	}
	assertAccountEqual(t, "tag51", acc, "ID.CO.QRIS.WWW", "9360091500001234567", "ID1020012345678", "UMI")
}

func TestBuild_RoundTrip_Dynamic(t *testing.T) {
	b := NewBuilder().
		Dynamic("15000").
		MerchantName("KOPI KITA").
		MerchantCity("BANDUNG").
		MCC("5812").
		AddQRISAccount("9360091500009999999", "ID1020099999999", "UKE")

	p := buildAndParse(t, b)

	if !p.IsDynamic() {
		t.Errorf("expected dynamic QR, got POIM %q", p.PointOfInitiationMethod)
	}
	if p.TransactionAmount != "15000" {
		t.Errorf("TransactionAmount: got %q, want %q", p.TransactionAmount, "15000")
	}
}

func TestBuild_RoundTrip_PSPAccount_Tag26(t *testing.T) {
	b := NewBuilder().
		Static().
		MerchantName("TOKO PSP").
		MerchantCity("SURABAYA").
		MCC("4812").
		AddPSPAccount("26", "ID.DANA.WWW", "936008990000000001", "002100000001", "UMI")

	p := buildAndParse(t, b)

	psps := p.PSPMerchantAccounts()
	if len(psps) != 1 {
		t.Fatalf("expected 1 PSP account, got %d", len(psps))
	}
	assertAccountEqual(t, "tag26", psps[0], "ID.DANA.WWW", "936008990000000001", "002100000001", "UMI")

	if _, ok := p.QRISMerchantAccount(); ok {
		t.Error("did not expect a tag 51 account")
	}
}

func TestBuild_RoundTrip_Tag26AndTag51(t *testing.T) {
	b := NewBuilder().
		Static().
		MerchantName("MULTI ACCT").
		MerchantCity("MEDAN").
		MCC("4812").
		AddPSPAccount("26", "ID.DANA.WWW", "936008990000000001", "002100000001", "UMI").
		AddQRISAccount("9360091500001234567", "ID1020012345678", "UMI")

	p := buildAndParse(t, b)

	// Tag 51 present.
	acc51, ok := p.QRISMerchantAccount()
	if !ok {
		t.Fatal("expected tag 51 account")
	}
	assertAccountEqual(t, "tag51", acc51, "ID.CO.QRIS.WWW", "9360091500001234567", "ID1020012345678", "UMI")

	// Tag 26 present.
	psps := p.PSPMerchantAccounts()
	if len(psps) != 1 {
		t.Fatalf("expected 1 PSP account, got %d", len(psps))
	}
	assertAccountEqual(t, "tag26", psps[0], "ID.DANA.WWW", "936008990000000001", "002100000001", "UMI")

	if len(p.MerchantAccountInfo) != 2 {
		t.Errorf("expected 2 merchant account entries, got %d", len(p.MerchantAccountInfo))
	}
}

func TestBuild_RoundTrip_AdditionalData(t *testing.T) {
	ad := &AdditionalData{
		BillNumber:           "INV-001",
		StoreLabel:           "STORE-7",
		TerminalLabel:        "TERM-1",
		PurposeOfTransaction: "PURCHASE",
	}
	b := NewBuilder().
		Static().
		MerchantName("WARUNG AD").
		MerchantCity("JAKARTA").
		MCC("4812").
		AddQRISAccount("9360091500001234567", "ID1020012345678", "UMI").
		SetAdditionalData(ad)

	p := buildAndParse(t, b)

	if p.AdditionalData == nil {
		t.Fatal("expected AdditionalData (tag 62)")
	}
	if p.AdditionalData.BillNumber != "INV-001" {
		t.Errorf("BillNumber: got %q", p.AdditionalData.BillNumber)
	}
	if p.AdditionalData.StoreLabel != "STORE-7" {
		t.Errorf("StoreLabel: got %q", p.AdditionalData.StoreLabel)
	}
	if p.AdditionalData.TerminalLabel != "TERM-1" {
		t.Errorf("TerminalLabel: got %q", p.AdditionalData.TerminalLabel)
	}
	if p.AdditionalData.PurposeOfTransaction != "PURCHASE" {
		t.Errorf("PurposeOfTransaction: got %q", p.AdditionalData.PurposeOfTransaction)
	}
	// Unset sub-tags should remain empty.
	if p.AdditionalData.MobileNumber != "" {
		t.Errorf("MobileNumber should be empty, got %q", p.AdditionalData.MobileNumber)
	}
}

// --- Tag ordering test ---

func TestBuild_TagOrdering(t *testing.T) {
	payload, err := NewBuilder().
		Dynamic("15000").
		MerchantName("ORDER TEST").
		MerchantCity("JAKARTA").
		PostalCode("12345").
		MCC("4812").
		AddPSPAccount("26", "ID.DANA.WWW", "936008990000000001", "002100000001", "UMI").
		AddQRISAccount("9360091500001234567", "ID1020012345678", "UMI").
		SetAdditionalData(&AdditionalData{StoreLabel: "S1"}).
		Build()
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}

	// Each tag header must appear, in this order, scanning left to right.
	wantOrder := []string{"00", "01", "26", "51", "52", "53", "54", "58", "59", "60", "61", "62", "6304"}
	pos := 0
	for _, tag := range wantOrder {
		idx := strings.Index(payload[pos:], tag)
		if idx < 0 {
			t.Fatalf("tag %q not found after position %d in %s", tag, pos, payload)
		}
		pos += idx + len(tag)
	}
}

// --- Error tests ---

func TestBuild_MissingRequiredFields_ListsAll(t *testing.T) {
	// Empty builder: nothing set at all.
	_, err := NewBuilder().Build()
	if err == nil {
		t.Fatal("expected error for empty builder")
	}

	msg := err.Error()
	wantSubstrings := []string{
		"point of initiation method",
		"merchant account",
		"merchant category code",
		"merchant name",
		"merchant city",
	}
	for _, want := range wantSubstrings {
		if !strings.Contains(msg, want) {
			t.Errorf("error message missing %q\nfull message: %s", want, msg)
		}
	}
}

func TestBuild_Dynamic_MissingAmount(t *testing.T) {
	// Dynamic with empty amount must be reported. We cannot reach this
	// via Dynamic("") alone since that sets POIM; verify the validator
	// flags the empty amount.
	b := NewBuilder().
		Dynamic("").
		MerchantName("NO AMOUNT").
		MerchantCity("JAKARTA").
		MCC("4812").
		AddQRISAccount("9360091500001234567", "ID1020012345678", "UMI")

	_, err := b.Build()
	if err == nil {
		t.Fatal("expected error for dynamic QR with no amount")
	}
	if !strings.Contains(err.Error(), "transaction amount") {
		t.Errorf("error should mention transaction amount, got: %s", err.Error())
	}
}

func TestBuild_StaticOverridesAmount(t *testing.T) {
	// Calling Static() after Dynamic() must clear the amount.
	p := buildAndParse(t, NewBuilder().
		Dynamic("99999").
		Static().
		MerchantName("FLIP").
		MerchantCity("JAKARTA").
		MCC("4812").
		AddQRISAccount("9360091500001234567", "ID1020012345678", "UMI"))

	if p.TransactionAmount != "" {
		t.Errorf("Static() should clear amount, got %q", p.TransactionAmount)
	}
	if !p.IsStatic() {
		t.Errorf("expected static, got POIM %q", p.PointOfInitiationMethod)
	}
}
