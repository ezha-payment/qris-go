package qris

import (
	"errors"
	"strings"
	"testing"
)

// validPayload returns a Payload that passes every validation rule.
// Tests mutate a single field to exercise one negative case at a time.
func validPayload() *Payload {
	return &Payload{
		PayloadFormatIndicator:  "01",
		PointOfInitiationMethod: "11",
		MerchantAccountInfo: map[string]MerchantAccount{
			"51": {
				GloballyUniqueIdentifier: "ID.CO.QRIS.WWW",
				MPAN:                     "9360091500001234567",
				MID:                      "ID1020012345678",
				MerchantCriteria:         "UMI",
			},
		},
		MerchantCategoryCode: "4812",
		TransactionCurrency:  "360",
		CountryCode:          "ID",
		MerchantName:         "WARUNG SAMPLE",
		MerchantCity:         "JAKARTA",
	}
}

// hasFieldError reports whether err is a *ValidationError containing a
// FieldError whose Field starts with the given prefix.
func hasFieldError(err error, fieldPrefix string) bool {
	var ve *ValidationError
	if !errors.As(err, &ve) {
		return false
	}
	for _, fe := range ve.Errors {
		if strings.HasPrefix(fe.Field, fieldPrefix) {
			return true
		}
	}
	return false
}

// assertValid fails if Validate returns an error.
func assertValid(t *testing.T, p *Payload) {
	t.Helper()
	if err := Validate(p); err != nil {
		t.Fatalf("expected valid payload, got: %v", err)
	}
}

// assertFieldError fails unless Validate flags the given field.
func assertFieldError(t *testing.T, p *Payload, fieldPrefix string) {
	t.Helper()
	err := Validate(p)
	if err == nil {
		t.Fatalf("expected validation error for %s, got nil", fieldPrefix)
	}
	if !hasFieldError(err, fieldPrefix) {
		t.Fatalf("expected error for field %s, got: %v", fieldPrefix, err)
	}
}

// --- Baseline ---

func TestValidate_ValidPayload(t *testing.T) {
	assertValid(t, validPayload())
}

// --- Rule 1: PayloadFormatIndicator ---

func TestValidate_PayloadFormatIndicator(t *testing.T) {
	p := validPayload()
	p.PayloadFormatIndicator = "02"
	assertFieldError(t, p, "PayloadFormatIndicator")
}

// --- Rule 2: PointOfInitiationMethod ---

func TestValidate_PointOfInitiationMethod_Valid(t *testing.T) {
	p := validPayload()
	p.PointOfInitiationMethod = "12"
	p.TransactionAmount = "15000"
	assertValid(t, p)
}

func TestValidate_PointOfInitiationMethod_Invalid(t *testing.T) {
	p := validPayload()
	p.PointOfInitiationMethod = "99"
	assertFieldError(t, p, "PointOfInitiationMethod")
}

// --- Rule 3: MerchantCategoryCode ---

func TestValidate_MerchantCategoryCode_Invalid(t *testing.T) {
	for _, bad := range []string{"481", "48123", "48AB", ""} {
		p := validPayload()
		p.MerchantCategoryCode = bad
		assertFieldError(t, p, "MerchantCategoryCode")
	}
}

// --- Rule 4: TransactionCurrency ---

func TestValidate_TransactionCurrency_Invalid(t *testing.T) {
	for _, bad := range []string{"36", "3600", "36A", ""} {
		p := validPayload()
		p.TransactionCurrency = bad
		assertFieldError(t, p, "TransactionCurrency")
	}
}

// --- Rule 5: TransactionAmount ---

func TestValidate_TransactionAmount_RequiredForDynamic(t *testing.T) {
	p := validPayload()
	p.PointOfInitiationMethod = "12"
	p.TransactionAmount = ""
	assertFieldError(t, p, "TransactionAmount")
}

func TestValidate_TransactionAmount_ValidDecimal(t *testing.T) {
	p := validPayload()
	p.PointOfInitiationMethod = "12"
	p.TransactionAmount = "15000.50"
	assertValid(t, p)
}

func TestValidate_TransactionAmount_Invalid(t *testing.T) {
	for _, bad := range []string{"abc", "1.2.3", "15,000", "12345678901234", "1."} {
		p := validPayload()
		p.PointOfInitiationMethod = "12"
		p.TransactionAmount = bad
		assertFieldError(t, p, "TransactionAmount")
	}
}

// --- Rule 6: CountryCode ---

func TestValidate_CountryCode_Invalid(t *testing.T) {
	for _, bad := range []string{"I", "IDN", "id", "1D"} {
		p := validPayload()
		p.CountryCode = bad
		assertFieldError(t, p, "CountryCode")
	}
}

// --- Rule 7: MerchantName ---

func TestValidate_MerchantName_TooLong(t *testing.T) {
	p := validPayload()
	p.MerchantName = strings.Repeat("A", 26)
	assertFieldError(t, p, "MerchantName")
}

func TestValidate_MerchantName_Empty(t *testing.T) {
	p := validPayload()
	p.MerchantName = ""
	assertFieldError(t, p, "MerchantName")
}

func TestValidate_MerchantName_NonPrintable(t *testing.T) {
	p := validPayload()
	p.MerchantName = "WARUNG\tX"
	assertFieldError(t, p, "MerchantName")
}

// --- Rule 8: MerchantCity ---

func TestValidate_MerchantCity_TooLong(t *testing.T) {
	p := validPayload()
	p.MerchantCity = strings.Repeat("B", 16)
	assertFieldError(t, p, "MerchantCity")
}

func TestValidate_MerchantCity_Valid(t *testing.T) {
	p := validPayload()
	p.MerchantCity = strings.Repeat("B", 15)
	assertValid(t, p)
}

// --- Rule 9: PostalCode ---

func TestValidate_PostalCode_OptionalEmpty(t *testing.T) {
	p := validPayload()
	p.PostalCode = ""
	assertValid(t, p)
}

func TestValidate_PostalCode_TooLong(t *testing.T) {
	p := validPayload()
	p.PostalCode = strings.Repeat("9", 11)
	assertFieldError(t, p, "PostalCode")
}

// --- Rule 10: MerchantAccountInfo presence ---

func TestValidate_MerchantAccounts_None(t *testing.T) {
	p := validPayload()
	p.MerchantAccountInfo = nil
	assertFieldError(t, p, "MerchantAccountInfo")
}

func TestValidate_MerchantAccounts_OutOfRangeTag(t *testing.T) {
	p := validPayload()
	p.MerchantAccountInfo = map[string]MerchantAccount{
		"47": {GloballyUniqueIdentifier: "ID.FOO.WWW"},
	}
	assertFieldError(t, p, "MerchantAccountInfo")
}

func TestValidate_MerchantAccounts_PSPTag26Valid(t *testing.T) {
	p := validPayload()
	p.MerchantAccountInfo = map[string]MerchantAccount{
		"26": {GloballyUniqueIdentifier: "ID.DANA.WWW", MerchantCriteria: "UKE"},
	}
	assertValid(t, p)
}

// --- Rule 11: GloballyUniqueIdentifier ---

func TestValidate_GUI_TooLong(t *testing.T) {
	p := validPayload()
	p.MerchantAccountInfo = map[string]MerchantAccount{
		"26": {GloballyUniqueIdentifier: strings.Repeat("X", 33)},
	}
	assertFieldError(t, p, "MerchantAccountInfo[26].GloballyUniqueIdentifier")
}

func TestValidate_GUI_Tag51WrongValue(t *testing.T) {
	p := validPayload()
	acc := p.MerchantAccountInfo["51"]
	acc.GloballyUniqueIdentifier = "ID.WRONG.WWW"
	p.MerchantAccountInfo["51"] = acc
	assertFieldError(t, p, "MerchantAccountInfo[51].GloballyUniqueIdentifier")
}

// --- Rule 12: MerchantCriteria ---

func TestValidate_MerchantCriteria_Invalid(t *testing.T) {
	p := validPayload()
	acc := p.MerchantAccountInfo["51"]
	acc.MerchantCriteria = "XXX"
	p.MerchantAccountInfo["51"] = acc
	assertFieldError(t, p, "MerchantAccountInfo[51].MerchantCriteria")
}

func TestValidate_MerchantCriteria_EmptyAllowed(t *testing.T) {
	p := validPayload()
	acc := p.MerchantAccountInfo["51"]
	acc.MerchantCriteria = ""
	p.MerchantAccountInfo["51"] = acc
	assertValid(t, p)
}

// --- Rule 13: AdditionalData ---

func TestValidate_AdditionalData_FieldTooLong(t *testing.T) {
	p := validPayload()
	p.AdditionalData = &AdditionalData{
		StoreLabel: strings.Repeat("S", 26),
	}
	assertFieldError(t, p, "AdditionalData.StoreLabel")
}

func TestValidate_AdditionalData_Valid(t *testing.T) {
	p := validPayload()
	p.AdditionalData = &AdditionalData{
		BillNumber:           "INV-001",
		PurposeOfTransaction: "PURCHASE",
		Raw:                  map[string]string{"09": "OK"},
	}
	assertValid(t, p)
}

func TestValidate_AdditionalData_RawTooLong(t *testing.T) {
	p := validPayload()
	p.AdditionalData = &AdditionalData{
		Raw: map[string]string{"09": strings.Repeat("R", 26)},
	}
	assertFieldError(t, p, "AdditionalData.Raw[09]")
}

// --- Aggregation: multiple simultaneous violations ---

func TestValidate_AggregatesMultipleErrors(t *testing.T) {
	p := validPayload()
	p.PayloadFormatIndicator = "99" // violation 1
	p.MerchantCategoryCode = "AB"   // violation 2
	p.MerchantName = ""             // violation 3

	err := Validate(p)
	if err == nil {
		t.Fatal("expected aggregated validation error")
	}

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if len(ve.Errors) != 3 {
		t.Fatalf("expected 3 field errors, got %d: %v", len(ve.Errors), ve.Errors)
	}
	for _, want := range []string{"PayloadFormatIndicator", "MerchantCategoryCode", "MerchantName"} {
		if !hasFieldError(err, want) {
			t.Errorf("missing expected field error %q in %v", want, err)
		}
	}
}

// --- errors.Is / errors.As compatibility ---

func TestValidationError_ErrorsIs(t *testing.T) {
	err := Validate(&Payload{}) // many violations
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrValidation) {
		t.Error("errors.Is(err, ErrValidation) should be true")
	}
}

func TestValidationError_ErrorsAs_FieldError(t *testing.T) {
	p := validPayload()
	p.MerchantName = ""
	err := Validate(p)

	var fe *FieldError
	if !errors.As(err, &fe) {
		t.Fatal("errors.As should extract a *FieldError")
	}
	if fe.Field != "MerchantName" {
		t.Errorf("extracted FieldError.Field = %q, want MerchantName", fe.Field)
	}
}

// --- Standalone Validate on a parsed Payload ---

func TestValidate_OnParsedPayload(t *testing.T) {
	// Build a valid payload, parse it back, then validate the parsed
	// result. This proves Validate works on Parser output, not only on
	// Builder-internal payloads.
	payload, err := NewBuilder().
		Static().
		MerchantName("WARUNG SAMPLE").
		MerchantCity("JAKARTA").
		MCC("4812").
		AddQRISAccount("9360091500001234567", "ID1020012345678", "UMI").
		Build()
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}

	parsed, err := Parse(payload)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	if err := Validate(parsed); err != nil {
		t.Errorf("Validate(parsed) should pass, got: %v", err)
	}
}

// --- Build integration: validation fires before CRC ---

func TestBuild_RejectsInvalidField(t *testing.T) {
	// MCC is non-empty (passes presence check) but not 4 digits, so the
	// failure must come from Validate, not the presence check.
	_, err := NewBuilder().
		Static().
		MerchantName("WARUNG SAMPLE").
		MerchantCity("JAKARTA").
		MCC("48").
		AddQRISAccount("9360091500001234567", "ID1020012345678", "UMI").
		Build()
	if err == nil {
		t.Fatal("expected Build to reject invalid MCC")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation, got: %v", err)
	}
}
