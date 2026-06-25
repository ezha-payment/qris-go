package qris

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ErrValidation is the sentinel wrapped by every ValidationError, so
// callers can test errors.Is(err, ErrValidation) regardless of which
// fields failed.
var ErrValidation = errors.New("qris: validation failed")

// FieldError describes a single field-level validation failure.
type FieldError struct {
	// Field is the name of the offending field (e.g. "MerchantName").
	Field string
	// Reason explains why the value is invalid.
	Reason string
}

// Error implements the error interface.
func (e *FieldError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}

// ValidationError aggregates one or more FieldError values produced by
// Validate. It implements error, satisfies errors.Is(err, ErrValidation)
// via Is, and exposes each underlying FieldError to errors.As via
// Unwrap.
type ValidationError struct {
	Errors []*FieldError
}

// Error implements the error interface, joining every field error.
func (e *ValidationError) Error() string {
	parts := make([]string, len(e.Errors))
	for i, fe := range e.Errors {
		parts[i] = fe.Error()
	}
	return fmt.Sprintf("qris: validation failed: %s", strings.Join(parts, "; "))
}

// Is reports whether target is ErrValidation, enabling
// errors.Is(err, ErrValidation).
func (e *ValidationError) Is(target error) bool {
	return target == ErrValidation
}

// Unwrap returns the individual field errors so that, for example,
// errors.As(err, &fe) with a *FieldError target succeeds.
func (e *ValidationError) Unwrap() []error {
	errs := make([]error, len(e.Errors))
	for i, fe := range e.Errors {
		errs[i] = fe
	}
	return errs
}

// FieldErrors returns the individual field errors.
func (e *ValidationError) FieldErrors() []*FieldError {
	return e.Errors
}

// Validate checks a Payload against ASPI/EMVCo-aligned rules and
// returns a *ValidationError aggregating every violation found, or nil
// if the Payload is valid.
//
// Validate is intentionally stricter than Parse: Parse is permissive so
// it can decode real-world payloads, whereas Validate enforces the
// constraints required to emit a conformant payload. Build calls
// Validate automatically before computing the CRC.
func Validate(p *Payload) error {
	v := &validator{}

	v.checkPayloadFormatIndicator(p.PayloadFormatIndicator)
	v.checkPointOfInitiationMethod(p.PointOfInitiationMethod)
	v.checkMerchantCategoryCode(p.MerchantCategoryCode)
	v.checkTransactionCurrency(p.TransactionCurrency)
	v.checkTransactionAmount(p.PointOfInitiationMethod, p.TransactionAmount)
	v.checkCountryCode(p.CountryCode)
	v.checkMerchantName(p.MerchantName)
	v.checkMerchantCity(p.MerchantCity)
	v.checkPostalCode(p.PostalCode)
	v.checkMerchantAccounts(p.MerchantAccountInfo)
	v.checkAdditionalData(p.AdditionalData)

	if len(v.errs) == 0 {
		return nil
	}
	return &ValidationError{Errors: v.errs}
}

// validator accumulates field errors across the rule checks.
type validator struct {
	errs []*FieldError
}

// add records a field violation.
func (v *validator) add(field, reason string) {
	v.errs = append(v.errs, &FieldError{Field: field, Reason: reason})
}

// --- Rule 1: PayloadFormatIndicator ---

func (v *validator) checkPayloadFormatIndicator(s string) {
	if s != "01" {
		v.add("PayloadFormatIndicator", fmt.Sprintf("must be %q, got %q", "01", s))
	}
}

// --- Rule 2: PointOfInitiationMethod ---

func (v *validator) checkPointOfInitiationMethod(s string) {
	if s != "11" && s != "12" {
		v.add("PointOfInitiationMethod", fmt.Sprintf("must be %q (static) or %q (dynamic), got %q", "11", "12", s))
	}
}

// --- Rule 3: MerchantCategoryCode ---

func (v *validator) checkMerchantCategoryCode(s string) {
	if len(s) != 4 || !isDigits(s) {
		v.add("MerchantCategoryCode", fmt.Sprintf("must be exactly 4 digits, got %q", s))
	}
}

// --- Rule 4: TransactionCurrency ---

func (v *validator) checkTransactionCurrency(s string) {
	if len(s) != 3 || !isDigits(s) {
		v.add("TransactionCurrency", fmt.Sprintf("must be exactly 3 digits, got %q", s))
	}
}

// --- Rule 5: TransactionAmount ---

func (v *validator) checkTransactionAmount(poi, amount string) {
	if poi == "12" && amount == "" {
		v.add("TransactionAmount", "required for dynamic QR (point of initiation \"12\")")
		return
	}
	if amount == "" {
		return // absent and not required.
	}
	if len(amount) > 13 {
		v.add("TransactionAmount", fmt.Sprintf("must be at most 13 characters, got %d", len(amount)))
	}
	if !isAmount(amount) {
		v.add("TransactionAmount", fmt.Sprintf("must be numeric with an optional single decimal point, got %q", amount))
	}
}

// --- Rule 6: CountryCode ---

func (v *validator) checkCountryCode(s string) {
	if len(s) != 2 || !isUpperAlpha(s) {
		v.add("CountryCode", fmt.Sprintf("must be exactly 2 uppercase ASCII letters, got %q", s))
	}
}

// --- Rule 7: MerchantName ---

func (v *validator) checkMerchantName(s string) {
	if len(s) < 1 || len(s) > 25 {
		v.add("MerchantName", fmt.Sprintf("must be 1-25 characters, got %d", len(s)))
	} else if !isASCIIPrintable(s) {
		v.add("MerchantName", "must contain only printable ASCII characters")
	}
}

// --- Rule 8: MerchantCity ---

func (v *validator) checkMerchantCity(s string) {
	if len(s) < 1 || len(s) > 15 {
		v.add("MerchantCity", fmt.Sprintf("must be 1-15 characters, got %d", len(s)))
	} else if !isASCIIPrintable(s) {
		v.add("MerchantCity", "must contain only printable ASCII characters")
	}
}

// --- Rule 9: PostalCode ---

func (v *validator) checkPostalCode(s string) {
	if s != "" && len(s) > 10 {
		v.add("PostalCode", fmt.Sprintf("must be at most 10 characters, got %d", len(s)))
	}
}

// --- Rules 10, 11, 12: MerchantAccountInfo ---

func (v *validator) checkMerchantAccounts(accounts map[string]MerchantAccount) {
	if len(accounts) == 0 {
		v.add("MerchantAccountInfo", "at least one merchant account (tag 26-45 or tag 51) is required")
		return
	}

	// Rule 10: at least one entry in the valid PSP range (26-45) or
	// the national QRIS tag (51).
	hasValid := false
	for tag := range accounts {
		if isPSPTag(tag) || tag == tagNationalQRIS {
			hasValid = true
			break
		}
	}
	if !hasValid {
		v.add("MerchantAccountInfo", "no merchant account in tag range 26-45 or tag 51")
	}

	// Rules 11 and 12 apply to every account present. Iterate tags in
	// sorted order so error output is deterministic.
	tags := make([]string, 0, len(accounts))
	for tag := range accounts {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	for _, tag := range tags {
		v.checkMerchantAccount(tag, accounts[tag])
	}
}

// checkMerchantAccount validates a single merchant account block.
func (v *validator) checkMerchantAccount(tag string, acc MerchantAccount) {
	field := "MerchantAccountInfo[" + tag + "]"

	// Rule 11: GUI length and, for tag 51, exact national value. The
	// national QRIS routing tag must carry "ID.CO.QRIS.WWW"; acquirer
	// or PSP specific GUIs belong in tags 26-50.
	gui := acc.GloballyUniqueIdentifier
	if len(gui) < 1 || len(gui) > 32 || !isASCIIPrintable(gui) {
		v.add(field+".GloballyUniqueIdentifier",
			fmt.Sprintf("must be 1-32 printable ASCII characters, got %q", gui))
	}
	if tag == tagNationalQRIS && gui != guiNationalQRIS {
		v.add(field+".GloballyUniqueIdentifier",
			fmt.Sprintf("tag 51 must use %q, got %q", guiNationalQRIS, gui))
	}

	// Rule 12: merchant criteria, if present, from the allowed set.
	if acc.MerchantCriteria != "" && !isValidCriteria(acc.MerchantCriteria) {
		v.add(field+".MerchantCriteria",
			fmt.Sprintf("must be one of UMI, UKE, UME, UBE, URE, got %q", acc.MerchantCriteria))
	}
}

// --- Rule 13: AdditionalData ---

func (v *validator) checkAdditionalData(ad *AdditionalData) {
	if ad == nil {
		return
	}
	fields := []struct {
		name  string
		value string
	}{
		{"BillNumber", ad.BillNumber},
		{"MobileNumber", ad.MobileNumber},
		{"StoreLabel", ad.StoreLabel},
		{"LoyaltyNumber", ad.LoyaltyNumber},
		{"ReferenceLabel", ad.ReferenceLabel},
		{"CustomerLabel", ad.CustomerLabel},
		{"TerminalLabel", ad.TerminalLabel},
		{"PurposeOfTransaction", ad.PurposeOfTransaction},
	}
	for _, f := range fields {
		if len(f.value) > 25 {
			v.add("AdditionalData."+f.name,
				fmt.Sprintf("must be at most 25 characters, got %d", len(f.value)))
		}
	}

	// Raw sub-tags are subject to the same per-field limit. Iterate in
	// sorted order for deterministic output.
	rawTags := make([]string, 0, len(ad.Raw))
	for tag := range ad.Raw {
		rawTags = append(rawTags, tag)
	}
	sort.Strings(rawTags)
	for _, tag := range rawTags {
		if len(ad.Raw[tag]) > 25 {
			v.add("AdditionalData.Raw["+tag+"]",
				fmt.Sprintf("must be at most 25 characters, got %d", len(ad.Raw[tag])))
		}
	}
}

// --- Shared predicates ---

// isDigits reports whether s is non-empty and all ASCII digits.
func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// isUpperAlpha reports whether s is non-empty and all ASCII A-Z.
func isUpperAlpha(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < 'A' || s[i] > 'Z' {
			return false
		}
	}
	return true
}

// isASCIIPrintable reports whether every byte of s is in the printable
// ASCII range (0x20 space through 0x7E tilde). Empty strings are
// considered printable; length is checked separately by each rule.
func isASCIIPrintable(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] > 0x7E {
			return false
		}
	}
	return true
}

// isAmount reports whether s is a valid transaction amount: one or more
// digits with at most one decimal point that, if present, has digits on
// both sides (e.g. "15000", "15000.50"). No sign, no thousands
// separators.
func isAmount(s string) bool {
	intPart, fracPart, hasDot := strings.Cut(s, ".")
	if !hasDot {
		return isDigits(s)
	}
	if strings.IndexByte(fracPart, '.') != -1 {
		return false // more than one decimal point.
	}
	return isDigits(intPart) && isDigits(fracPart)
}

// isPSPTag reports whether tag is in the PSP range 26-45 inclusive.
func isPSPTag(tag string) bool {
	n, err := strconv.Atoi(tag)
	if err != nil {
		return false
	}
	return n >= 26 && n <= 45
}

// isValidCriteria reports whether c is one of the allowed merchant
// criteria codes (see the Criteria* constants).
func isValidCriteria(c string) bool {
	switch MerchantCriteria(c) {
	case CriteriaUMI, CriteriaUKE, CriteriaUME, CriteriaUBE, CriteriaURE:
		return true
	default:
		return false
	}
}
