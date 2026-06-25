package qris

import (
	"errors"
	"strings"
	"testing"
)

// compliantStatic loads a static fixture whose tag 51 carries the
// national QRIS GUI, so it passes strict validation and can be
// converted. It also carries a tag 62 block (store + terminal labels),
// which the option tests rely on.
func compliantStatic(t *testing.T) string {
	t.Helper()
	return readTestFile(t, "testdata/valid/static_with_additional.txt")
}

// --- Happy path ---

func TestConvert_HappyPath(t *testing.T) {
	src := compliantStatic(t)

	orig, err := Parse(src)
	if err != nil {
		t.Fatalf("parse source: %v", err)
	}
	if !orig.IsStatic() {
		t.Fatalf("fixture precondition: expected static source, got POI %q", orig.PointOfInitiationMethod)
	}

	out, err := ConvertStaticToDynamic(src, "25000")
	if err != nil {
		t.Fatalf("ConvertStaticToDynamic: %v", err)
	}

	got, err := Parse(out)
	if err != nil {
		t.Fatalf("parse converted: %v\noutput: %s", err, out)
	}

	// Now dynamic with the embedded amount.
	if !got.IsDynamic() {
		t.Errorf("expected dynamic, got POI %q", got.PointOfInitiationMethod)
	}
	if got.TransactionAmount != "25000" {
		t.Errorf("TransactionAmount: got %q, want %q", got.TransactionAmount, "25000")
	}

	// All non-amount fields preserved.
	if got.MerchantName != orig.MerchantName {
		t.Errorf("MerchantName changed: %q -> %q", orig.MerchantName, got.MerchantName)
	}
	if got.MerchantCity != orig.MerchantCity {
		t.Errorf("MerchantCity changed: %q -> %q", orig.MerchantCity, got.MerchantCity)
	}
	if got.MerchantCategoryCode != orig.MerchantCategoryCode {
		t.Errorf("MCC changed: %q -> %q", orig.MerchantCategoryCode, got.MerchantCategoryCode)
	}
	if got.PostalCode != orig.PostalCode {
		t.Errorf("PostalCode changed: %q -> %q", orig.PostalCode, got.PostalCode)
	}
	if got.CountryCode != orig.CountryCode {
		t.Errorf("CountryCode changed: %q -> %q", orig.CountryCode, got.CountryCode)
	}
	if len(got.MerchantAccountInfo) != len(orig.MerchantAccountInfo) {
		t.Errorf("merchant account count changed: %d -> %d",
			len(orig.MerchantAccountInfo), len(got.MerchantAccountInfo))
	}
	for tag, want := range orig.MerchantAccountInfo {
		gotAcc, ok := got.MerchantAccountInfo[tag]
		if !ok {
			t.Errorf("merchant account tag %s dropped", tag)
			continue
		}
		if gotAcc.GloballyUniqueIdentifier != want.GloballyUniqueIdentifier ||
			gotAcc.MPAN != want.MPAN || gotAcc.MID != want.MID ||
			gotAcc.MerchantCriteria != want.MerchantCriteria {
			t.Errorf("merchant account tag %s changed: %+v -> %+v", tag, want, gotAcc)
		}
	}

	// CRC must be present and valid (Parse already verified it).
	if got.CRC == "" {
		t.Error("converted payload has no CRC")
	}
}

// --- Error: already dynamic ---

func TestConvert_AlreadyDynamic(t *testing.T) {
	dyn := readTestFile(t, "testdata/valid/dynamic_with_amount.txt")

	_, err := ConvertStaticToDynamic(dyn, "25000")
	if !errors.Is(err, ErrAlreadyDynamic) {
		t.Fatalf("expected ErrAlreadyDynamic, got: %v", err)
	}
}

// --- Error: parse failure ---

func TestConvert_ParseError(t *testing.T) {
	_, err := ConvertStaticToDynamic("not-a-qris-payload", "25000")
	if err == nil {
		t.Fatal("expected parse error")
	}
	if errors.Is(err, ErrAlreadyDynamic) {
		t.Fatal("should not be ErrAlreadyDynamic")
	}
	// The underlying parse sentinel must be unwrappable.
	if !errors.Is(err, ErrMissingCRCTag) && !errors.Is(err, ErrPayloadTooShort) &&
		!errors.Is(err, ErrInvalidCRC) && !errors.Is(err, ErrInvalidTLVFormat) {
		t.Errorf("expected a wrapped parse error, got: %v", err)
	}
}

// --- Error: source fails validation ---

func TestConvert_SourceValidationError(t *testing.T) {
	// The anonymized ZIPay fixture parses (Parser is permissive) but its
	// tag 51 GUI is "ID.ZIPAY.WWW", which violates strict rule 11. The
	// conversion must surface this as a wrapped validation error.
	src := readTestFile(t, "testdata/valid/zipay_static_anonymized.txt")

	_, err := ConvertStaticToDynamic(src, "25000")
	if err == nil {
		t.Fatal("expected validation error for non-conformant source")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected wrapped ErrValidation, got: %v", err)
	}
}

// --- Error: invalid amount ---

func TestConvert_InvalidAmount(t *testing.T) {
	src := compliantStatic(t)

	for _, amount := range []string{"abc", "12345678901234", "1.2.3"} {
		_, err := ConvertStaticToDynamic(src, amount)
		if err == nil {
			t.Errorf("amount %q: expected error", amount)
			continue
		}
		if !errors.Is(err, ErrValidation) {
			t.Errorf("amount %q: expected ErrValidation, got: %v", amount, err)
		}
	}
}

func TestConvert_EmptyAmount(t *testing.T) {
	src := compliantStatic(t)

	_, err := ConvertStaticToDynamic(src, "")
	if err == nil {
		t.Fatal("expected error for empty amount")
	}
}

// --- Options: tag 62 ---

func TestConvert_WithReferenceAndTerminalLabel(t *testing.T) {
	src := compliantStatic(t)

	out, err := ConvertStaticToDynamic(src, "25000",
		WithReferenceLabel("REF-123"),
		WithTerminalLabel("TERM-9"),
	)
	if err != nil {
		t.Fatalf("ConvertStaticToDynamic: %v", err)
	}

	got, err := Parse(out)
	if err != nil {
		t.Fatalf("parse converted: %v", err)
	}
	if got.AdditionalData == nil {
		t.Fatal("expected tag 62 AdditionalData in output")
	}
	if got.AdditionalData.ReferenceLabel != "REF-123" {
		t.Errorf("ReferenceLabel: got %q, want %q", got.AdditionalData.ReferenceLabel, "REF-123")
	}
	if got.AdditionalData.TerminalLabel != "TERM-9" {
		t.Errorf("TerminalLabel: got %q, want %q", got.AdditionalData.TerminalLabel, "TERM-9")
	}
}

func TestConvert_OptionsOverrideButPreserveOtherSubFields(t *testing.T) {
	// The compliant fixture carries a store label and terminal label in
	// tag 62. Override only the store label; the terminal label must
	// survive untouched.
	src := compliantStatic(t)

	orig, err := Parse(src)
	if err != nil {
		t.Fatalf("parse source: %v", err)
	}
	if orig.AdditionalData == nil || orig.AdditionalData.TerminalLabel == "" {
		t.Fatalf("fixture precondition: expected an original terminal label, got %+v", orig.AdditionalData)
	}
	origTerminal := orig.AdditionalData.TerminalLabel

	out, err := ConvertStaticToDynamic(src, "25000", WithStoreLabel("NEW-STORE"))
	if err != nil {
		t.Fatalf("ConvertStaticToDynamic: %v", err)
	}
	got, err := Parse(out)
	if err != nil {
		t.Fatalf("parse converted: %v", err)
	}
	if got.AdditionalData.StoreLabel != "NEW-STORE" {
		t.Errorf("StoreLabel not overridden: got %q", got.AdditionalData.StoreLabel)
	}
	if got.AdditionalData.TerminalLabel != origTerminal {
		t.Errorf("TerminalLabel not preserved: got %q, want %q",
			got.AdditionalData.TerminalLabel, origTerminal)
	}
}

func TestConvert_DoesNotMutateSource(t *testing.T) {
	// Converting with an override must not mutate the original Payload's
	// AdditionalData (guards the Raw-map copy in mergeAdditionalData).
	src := compliantStatic(t)
	orig, _ := Parse(src)
	before := ""
	if orig.AdditionalData != nil {
		before = orig.AdditionalData.StoreLabel
	}

	if _, err := ConvertStaticToDynamic(src, "25000", WithStoreLabel("MUTANT")); err != nil {
		t.Fatalf("convert: %v", err)
	}

	// Re-parse the untouched source and confirm it is unchanged.
	reParsed, _ := Parse(src)
	if reParsed.AdditionalData != nil && reParsed.AdditionalData.StoreLabel != before {
		t.Errorf("source store label mutated: %q -> %q", before, reParsed.AdditionalData.StoreLabel)
	}
}

// --- Determinism ---

func TestConvert_Deterministic(t *testing.T) {
	src := compliantStatic(t)

	opts := []ConvertOption{
		WithReferenceLabel("REF-1"),
		WithTerminalLabel("T-1"),
		WithStoreLabel("S-1"),
	}

	out1, err1 := ConvertStaticToDynamic(src, "25000", opts...)
	out2, err2 := ConvertStaticToDynamic(src, "25000", opts...)
	if err1 != nil || err2 != nil {
		t.Fatalf("convert errors: %v / %v", err1, err2)
	}
	if out1 != out2 {
		t.Errorf("non-deterministic output:\n%s\n%s", out1, out2)
	}
	if !strings.Contains(out1, "54") {
		t.Error("expected an amount tag in output")
	}
}
