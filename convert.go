package qris

import (
	"errors"
	"fmt"
	"maps"
)

// ErrAlreadyDynamic is returned by ConvertStaticToDynamic when the input
// payload is already a dynamic QR (point of initiation method "12").
var ErrAlreadyDynamic = errors.New("qris: payload is already dynamic")

// ConvertOption customizes a ConvertStaticToDynamic call.
type ConvertOption func(*convertConfig)

// convertConfig holds the optional tag 62 overrides for a conversion.
// Each field is a pointer so that "not provided" is distinguishable
// from "set to empty string".
type convertConfig struct {
	referenceLabel *string
	terminalLabel  *string
	storeLabel     *string
}

// WithReferenceLabel sets the additional data reference label
// (tag 62, sub-tag 05) on the converted payload, overriding any value
// carried by the original.
func WithReferenceLabel(ref string) ConvertOption {
	return func(c *convertConfig) { c.referenceLabel = &ref }
}

// WithTerminalLabel sets the additional data terminal label
// (tag 62, sub-tag 07) on the converted payload, overriding any value
// carried by the original.
func WithTerminalLabel(terminal string) ConvertOption {
	return func(c *convertConfig) { c.terminalLabel = &terminal }
}

// WithStoreLabel sets the additional data store label
// (tag 62, sub-tag 03) on the converted payload, overriding any value
// carried by the original.
func WithStoreLabel(store string) ConvertOption {
	return func(c *convertConfig) { c.storeLabel = &store }
}

// ConvertStaticToDynamic turns a static QRIS payload into a dynamic one
// carrying the given transaction amount.
//
// This is the acquirer point-of-sale use case: a merchant displays a
// static QR, the customer's purchase amount is known at checkout, and
// the POS embeds that amount into a fresh dynamic payload. Every field
// of the original is preserved; only the point of initiation method
// (tag 01 becomes "12"), the amount (tag 54), and any additional data
// overrides (tag 62) change. The CRC is recomputed.
//
// The conversion runs Parser -> modify -> Builder internally and never
// manipulates the payload string directly. It returns:
//   - a wrapped parse error if staticPayload cannot be decoded,
//   - ErrAlreadyDynamic if the input is already a dynamic QR,
//   - a wrapped *ValidationError if the original or the resulting
//     payload (including the amount) is invalid.
func ConvertStaticToDynamic(staticPayload string, amount string, opts ...ConvertOption) (string, error) {
	cfg := &convertConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// Step 1: Parse the input payload.
	p, err := Parse(staticPayload)
	if err != nil {
		return "", fmt.Errorf("qris: convert: parsing payload: %w", err)
	}

	// Step 2: Reject payloads that are already dynamic.
	if p.IsDynamic() {
		return "", ErrAlreadyDynamic
	}

	// Step 3: Validate the original. This both confirms it is a
	// well-formed static payload and surfaces any field problems before
	// we build on top of it.
	if err := Validate(p); err != nil {
		return "", fmt.Errorf("qris: convert: invalid source payload: %w", err)
	}

	// Step 4: Rebuild via the Builder, switching to dynamic and merging
	// additional data. Field assignments are direct (this file is in
	// package qris) so that arbitrary merchant accounts and their Raw
	// sub-tags are preserved exactly; the public setters cannot carry
	// Raw sub-tags.
	b := NewBuilder()
	b.payloadFormatIndicator = p.PayloadFormatIndicator
	b.pointOfInitiationMethod = poiDynamic
	b.transactionAmount = amount
	b.merchantAccounts = p.MerchantAccountInfo
	b.merchantCategoryCode = p.MerchantCategoryCode
	b.transactionCurrency = p.TransactionCurrency
	b.countryCode = p.CountryCode
	b.merchantName = p.MerchantName
	b.merchantCity = p.MerchantCity
	b.postalCode = p.PostalCode
	b.additionalData = mergeAdditionalData(p.AdditionalData, cfg)

	// Step 5: Build. This recomputes every length prefix and the CRC,
	// and validates the result (including the amount format).
	out, err := b.Build()
	if err != nil {
		return "", fmt.Errorf("qris: convert: building dynamic payload: %w", err)
	}
	return out, nil
}

// mergeAdditionalData produces the tag 62 block for the converted
// payload: it starts from the original (if any), then applies whichever
// option overrides were provided. The original's Raw sub-tags and other
// fields are preserved. Returns nil when there is nothing to emit.
func mergeAdditionalData(orig *AdditionalData, cfg *convertConfig) *AdditionalData {
	if orig == nil && cfg.referenceLabel == nil && cfg.terminalLabel == nil && cfg.storeLabel == nil {
		return nil
	}

	ad := &AdditionalData{}
	if orig != nil {
		*ad = *orig
		// Copy the Raw map so the source Payload is never mutated.
		if orig.Raw != nil {
			ad.Raw = make(map[string]string, len(orig.Raw))
			maps.Copy(ad.Raw, orig.Raw)
		}
	}

	if cfg.referenceLabel != nil {
		ad.ReferenceLabel = *cfg.referenceLabel
	}
	if cfg.terminalLabel != nil {
		ad.TerminalLabel = *cfg.terminalLabel
	}
	if cfg.storeLabel != nil {
		ad.StoreLabel = *cfg.storeLabel
	}
	return ad
}
