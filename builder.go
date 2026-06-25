package qris

import (
	"fmt"
	"sort"
	"strings"
)

// Default values applied by the Builder when the caller does not set
// them explicitly. These match the only values valid for QRIS today.
const (
	defaultPayloadFormatIndicator = "01"
	defaultTransactionCurrency    = "360" // ISO 4217 numeric for IDR.
	defaultCountryCode            = "ID"  // ISO 3166-1 alpha-2 for Indonesia.

	// poiStatic and poiDynamic are the tag 01 values for static and
	// dynamic QR respectively.
	poiStatic  = "11"
	poiDynamic = "12"

	// guiNationalQRIS is the Globally Unique Identifier for the
	// national QRIS routing entry carried in tag 51.
	guiNationalQRIS = "ID.CO.QRIS.WWW"

	// tagNationalQRIS is the Merchant Account Information tag used for
	// the national QRIS routing identity.
	tagNationalQRIS = "51"
)

// Builder constructs valid QRIS payload strings from struct input.
//
// Build() always recomputes the CRC and every TLV length prefix, so
// callers never set those manually. A Builder is created with
// NewBuilder, configured through chainable setters, and finalized with
// Build:
//
//	payload, err := qris.NewBuilder().
//		Static().
//		MerchantName("WARUNG SAMPLE").
//		MerchantCity("JAKARTA").
//		MCC("4812").
//		AddQRISAccount("9360091500001234567", "ID1020012345678", "UMI").
//		Build()
//
// The output is guaranteed to round-trip through Parse: parsing a
// Builder's output yields an equivalent Payload.
type Builder struct {
	payloadFormatIndicator  string
	pointOfInitiationMethod string
	merchantAccounts        map[string]MerchantAccount
	merchantCategoryCode    string
	transactionCurrency     string
	transactionAmount       string
	countryCode             string
	merchantName            string
	merchantCity            string
	postalCode              string
	additionalData          *AdditionalData
}

// NewBuilder returns a Builder pre-populated with the QRIS defaults for
// payload format indicator (tag 00), transaction currency (tag 53), and
// country code (tag 58). Any of these may be overridden by the
// corresponding setter.
func NewBuilder() *Builder {
	return &Builder{
		payloadFormatIndicator: defaultPayloadFormatIndicator,
		transactionCurrency:    defaultTransactionCurrency,
		countryCode:            defaultCountryCode,
		merchantAccounts:       make(map[string]MerchantAccount),
	}
}

// Static marks the payload as a static QR (tag 01 = "11"), i.e. one
// without an embedded amount. It clears any previously set amount.
func (b *Builder) Static() *Builder {
	b.pointOfInitiationMethod = poiStatic
	b.transactionAmount = ""
	return b
}

// Dynamic marks the payload as a dynamic QR (tag 01 = "12") carrying
// the given transaction amount (tag 54).
func (b *Builder) Dynamic(amount string) *Builder {
	b.pointOfInitiationMethod = poiDynamic
	b.transactionAmount = amount
	return b
}

// MerchantName sets tag 59. Required.
func (b *Builder) MerchantName(name string) *Builder {
	b.merchantName = name
	return b
}

// MerchantCity sets tag 60. Required.
func (b *Builder) MerchantCity(city string) *Builder {
	b.merchantCity = city
	return b
}

// PostalCode sets tag 61. Optional.
func (b *Builder) PostalCode(code string) *Builder {
	b.postalCode = code
	return b
}

// MCC sets the Merchant Category Code (tag 52). Required.
func (b *Builder) MCC(mcc string) *Builder {
	b.merchantCategoryCode = mcc
	return b
}

// Currency sets the transaction currency (tag 53). Defaults to "360".
func (b *Builder) Currency(currency string) *Builder {
	b.transactionCurrency = currency
	return b
}

// Country sets the country code (tag 58). Defaults to "ID".
func (b *Builder) Country(country string) *Builder {
	b.countryCode = country
	return b
}

// AddQRISAccount adds the national QRIS routing entry as tag 51, using
// the standard GUI "ID.CO.QRIS.WWW". A payload needs at least one
// merchant account (this or AddPSPAccount).
func (b *Builder) AddQRISAccount(mpan, mid, criteria string) *Builder {
	b.merchantAccounts[tagNationalQRIS] = MerchantAccount{
		GloballyUniqueIdentifier: guiNationalQRIS,
		MPAN:                     mpan,
		MID:                      mid,
		MerchantCriteria:         criteria,
	}
	return b
}

// AddPSPAccount adds a Payment Service Provider specific merchant
// account at the given tag (one of 26-51) with the given GUI. A payload
// needs at least one merchant account (this or AddQRISAccount).
func (b *Builder) AddPSPAccount(tag, gui, mpan, mid, criteria string) *Builder {
	b.merchantAccounts[tag] = MerchantAccount{
		GloballyUniqueIdentifier: gui,
		MPAN:                     mpan,
		MID:                      mid,
		MerchantCriteria:         criteria,
	}
	return b
}

// SetAdditionalData sets the tag 62 additional data block. Optional.
func (b *Builder) SetAdditionalData(ad *AdditionalData) *Builder {
	b.additionalData = ad
	return b
}

// Build assembles the configured fields into a complete QRIS payload
// string, computing every TLV length prefix and the trailing CRC.
//
// All required fields are validated up front: if any are missing, the
// returned error lists all of them rather than failing on the first.
func (b *Builder) Build() (string, error) {
	if err := b.validate(); err != nil {
		return "", err
	}

	// Enforce the stricter ASPI/EMVCo rules on the assembled payload
	// before computing the CRC. The presence check above gives
	// builder-specific guidance; Validate covers value correctness.
	if err := Validate(b.toPayload()); err != nil {
		return "", err
	}

	var sb strings.Builder

	// Tags 00 and 01.
	sb.WriteString(encodeTLV("00", b.payloadFormatIndicator))
	sb.WriteString(encodeTLV("01", b.pointOfInitiationMethod))

	// Tags 26-51 in ascending numeric order. Tag strings are all two
	// digits, so lexical order equals numeric order.
	tags := make([]string, 0, len(b.merchantAccounts))
	for tag := range b.merchantAccounts {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	for _, tag := range tags {
		sb.WriteString(encodeTLV(tag, encodeMerchantAccount(b.merchantAccounts[tag])))
	}

	// Tag 52.
	sb.WriteString(encodeTLV("52", b.merchantCategoryCode))
	// Tag 53.
	sb.WriteString(encodeTLV("53", b.transactionCurrency))
	// Tag 54 only for dynamic QR.
	if b.transactionAmount != "" {
		sb.WriteString(encodeTLV("54", b.transactionAmount))
	}
	// Tags 58, 59, 60.
	sb.WriteString(encodeTLV("58", b.countryCode))
	sb.WriteString(encodeTLV("59", b.merchantName))
	sb.WriteString(encodeTLV("60", b.merchantCity))
	// Tag 61 optional.
	if b.postalCode != "" {
		sb.WriteString(encodeTLV("61", b.postalCode))
	}
	// Tag 62 optional.
	if ad := encodeAdditionalData(b.additionalData); ad != "" {
		sb.WriteString(encodeTLV("62", ad))
	}

	// Tag 63: append "6304" and the CRC computed over everything
	// before the CRC value, matching the parser's expectation.
	dataNoCRC := sb.String() + "6304"
	crc := formatCRC(computeCRC([]byte(dataNoCRC)))
	return dataNoCRC + crc, nil
}

// toPayload assembles the Builder's current state into a Payload for
// validation. It mirrors what Build serializes; the empty merchant
// account map is left as-is so Validate can flag its absence.
func (b *Builder) toPayload() *Payload {
	return &Payload{
		PayloadFormatIndicator:  b.payloadFormatIndicator,
		PointOfInitiationMethod: b.pointOfInitiationMethod,
		MerchantAccountInfo:     b.merchantAccounts,
		MerchantCategoryCode:    b.merchantCategoryCode,
		TransactionCurrency:     b.transactionCurrency,
		TransactionAmount:       b.transactionAmount,
		CountryCode:             b.countryCode,
		MerchantName:            b.merchantName,
		MerchantCity:            b.merchantCity,
		PostalCode:              b.postalCode,
		AdditionalData:          b.additionalData,
	}
}

// validate checks all required fields and returns a single error
// enumerating every missing one, or nil if the Builder is complete.
func (b *Builder) validate() error {
	var missing []string

	if b.pointOfInitiationMethod == "" {
		missing = append(missing, "point of initiation method (call Static or Dynamic)")
	}
	if b.pointOfInitiationMethod == poiDynamic && b.transactionAmount == "" {
		missing = append(missing, "transaction amount (required for dynamic QR)")
	}
	if len(b.merchantAccounts) == 0 {
		missing = append(missing, "merchant account (call AddQRISAccount or AddPSPAccount)")
	}
	if b.merchantCategoryCode == "" {
		missing = append(missing, "merchant category code (call MCC)")
	}
	if b.merchantName == "" {
		missing = append(missing, "merchant name (call MerchantName)")
	}
	if b.merchantCity == "" {
		missing = append(missing, "merchant city (call MerchantCity)")
	}

	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("qris: cannot build payload, missing required fields: %s",
		strings.Join(missing, "; "))
}

// encodeTLV encodes a single Tag-Length-Value triplet: the 2-character
// tag, the value length as a zero-padded 2-digit decimal, then the
// value. It is the inverse of the parser's tokenize step.
func encodeTLV(tag, value string) string {
	return fmt.Sprintf("%s%02d%s", tag, len(value), value)
}

// encodeMerchantAccount serializes a MerchantAccount into the nested
// TLV value for a tag 26-51 block, mirroring parseMerchantAccount.
// Empty modeled sub-tags are omitted; Raw entries are appended in
// ascending sub-tag order for deterministic output.
func encodeMerchantAccount(acc MerchantAccount) string {
	var sb strings.Builder
	if acc.GloballyUniqueIdentifier != "" {
		sb.WriteString(encodeTLV("00", acc.GloballyUniqueIdentifier))
	}
	if acc.MPAN != "" {
		sb.WriteString(encodeTLV("01", acc.MPAN))
	}
	if acc.MID != "" {
		sb.WriteString(encodeTLV("02", acc.MID))
	}
	if acc.MerchantCriteria != "" {
		sb.WriteString(encodeTLV("03", acc.MerchantCriteria))
	}
	sb.WriteString(encodeRaw(acc.Raw))
	return sb.String()
}

// encodeAdditionalData serializes an AdditionalData block into the
// nested TLV value for tag 62, mirroring parseAdditionalData. Returns
// an empty string for a nil or all-empty block so the caller can omit
// tag 62 entirely.
func encodeAdditionalData(ad *AdditionalData) string {
	if ad == nil {
		return ""
	}
	var sb strings.Builder
	if ad.BillNumber != "" {
		sb.WriteString(encodeTLV("01", ad.BillNumber))
	}
	if ad.MobileNumber != "" {
		sb.WriteString(encodeTLV("02", ad.MobileNumber))
	}
	if ad.StoreLabel != "" {
		sb.WriteString(encodeTLV("03", ad.StoreLabel))
	}
	if ad.LoyaltyNumber != "" {
		sb.WriteString(encodeTLV("04", ad.LoyaltyNumber))
	}
	if ad.ReferenceLabel != "" {
		sb.WriteString(encodeTLV("05", ad.ReferenceLabel))
	}
	if ad.CustomerLabel != "" {
		sb.WriteString(encodeTLV("06", ad.CustomerLabel))
	}
	if ad.TerminalLabel != "" {
		sb.WriteString(encodeTLV("07", ad.TerminalLabel))
	}
	if ad.PurposeOfTransaction != "" {
		sb.WriteString(encodeTLV("08", ad.PurposeOfTransaction))
	}
	sb.WriteString(encodeRaw(ad.Raw))
	return sb.String()
}

// encodeRaw serializes a Raw sub-tag map in ascending tag order for
// deterministic output.
func encodeRaw(raw map[string]string) string {
	if len(raw) == 0 {
		return ""
	}
	tags := make([]string, 0, len(raw))
	for tag := range raw {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	var sb strings.Builder
	for _, tag := range tags {
		sb.WriteString(encodeTLV(tag, raw[tag]))
	}
	return sb.String()
}
