package qris

// The package-level documentation lives in doc.go.

// Payload represents a parsed QRIS payment payload.
//
// QRIS payloads are TLV-encoded (Tag-Length-Value) strings ending
// with a CRC-16/CCITT-FALSE checksum.
type Payload struct {
	// PayloadFormatIndicator is tag 00. Always "01" for QRIS.
	PayloadFormatIndicator string

	// PointOfInitiationMethod is tag 01.
	// "11" = static QR (no amount), "12" = dynamic QR (with amount).
	PointOfInitiationMethod string

	// MerchantAccountInfo contains entries for tags 26 through 51.
	//
	// Tags 26-45 carry Payment Service Provider (PSP) specific
	// merchant data (e.g., DANA, GoPay, ShopeePay). Tag 51 carries
	// the national QRIS routing identity. A given payload may have
	// only 26 (or other 26-45 range), only 51, or both.
	//
	// The map key is the tag string ("26", "27", ..., "51") so
	// consumers can inspect exactly which slot was populated.
	MerchantAccountInfo map[string]MerchantAccount

	// MerchantCategoryCode is tag 52. ISO 18245 4-digit code.
	MerchantCategoryCode string

	// TransactionCurrency is tag 53. ISO 4217 numeric code.
	// "360" = Indonesian Rupiah (IDR).
	TransactionCurrency string

	// TransactionAmount is tag 54.
	// Empty for static QR, required for dynamic QR.
	TransactionAmount string

	// CountryCode is tag 58. ISO 3166-1 alpha-2 code.
	CountryCode string

	// MerchantName is tag 59. Maximum 25 characters.
	MerchantName string

	// MerchantCity is tag 60. Maximum 15 characters.
	MerchantCity string

	// PostalCode is tag 61. Optional.
	PostalCode string

	// AdditionalData contains parsed contents of tag 62.
	AdditionalData *AdditionalData

	// CRC is tag 63. 4-character uppercase hexadecimal
	// CRC-16/CCITT-FALSE checksum.
	CRC string
}

// MerchantAccount represents a single Merchant Account Information
// entry (one of tags 26-51).
type MerchantAccount struct {
	// GloballyUniqueIdentifier is sub-tag 00 within the merchant
	// account info block. Examples:
	//   - "ID.CO.QRIS.WWW"   (national QRIS switching)
	//   - "ID.DANA.WWW"      (DANA wallet)
	//   - "ID.LINKAJA.WWW"   (LinkAja)
	//   - "ID.SHOPEEPAY.WWW" (ShopeePay)
	GloballyUniqueIdentifier string

	// MPAN is the Merchant Primary Account Number (sub-tag 01).
	// Equivalent role to card PAN in card payments.
	// Typically 16-19 digits.
	MPAN string

	// MID is the Merchant ID (sub-tag 02), assigned by the
	// acquirer or PSP. Different from MPAN.
	MID string

	// MerchantCriteria is sub-tag 03. Common values:
	//   - "UMI" : Usaha Mikro (Micro)
	//   - "UKE" : Usaha Kecil (Small)
	//   - "UME" : Usaha Menengah (Medium)
	//   - "UBE" : Usaha Besar (Large)
	//   - "URE" : Regular (non-classified)
	MerchantCriteria string

	// Raw holds any sub-tags not explicitly modeled above,
	// keyed by sub-tag string. Useful for forward-compatibility
	// with PSP-specific extensions.
	Raw map[string]string
}

// AdditionalData represents the parsed content of tag 62.
type AdditionalData struct {
	BillNumber           string // sub-tag 01
	MobileNumber         string // sub-tag 02
	StoreLabel           string // sub-tag 03
	LoyaltyNumber        string // sub-tag 04
	ReferenceLabel       string // sub-tag 05
	CustomerLabel        string // sub-tag 06
	TerminalLabel        string // sub-tag 07
	PurposeOfTransaction string // sub-tag 08

	// Raw holds any sub-tags not explicitly modeled.
	Raw map[string]string
}

// IsStatic reports whether this is a static QR (no amount embedded).
func (p *Payload) IsStatic() bool {
	return p.PointOfInitiationMethod == "11"
}

// IsDynamic reports whether this is a dynamic QR (with amount).
func (p *Payload) IsDynamic() bool {
	return p.PointOfInitiationMethod == "12"
}

// QRISMerchantAccount returns the national QRIS routing entry
// (tag 51, with GUI "ID.CO.QRIS.WWW") if present.
//
// This is the most useful entry for QRIS-compliant acquirer
// integration: MPAN and NMID flow through here.
func (p *Payload) QRISMerchantAccount() (MerchantAccount, bool) {
	if p.MerchantAccountInfo == nil {
		return MerchantAccount{}, false
	}
	if acc, ok := p.MerchantAccountInfo["51"]; ok {
		return acc, true
	}
	return MerchantAccount{}, false
}

// PSPMerchantAccounts returns all PSP-specific entries (tags 26-45).
// Order is not guaranteed.
func (p *Payload) PSPMerchantAccounts() []MerchantAccount {
	if p.MerchantAccountInfo == nil {
		return nil
	}
	result := make([]MerchantAccount, 0)
	for tag, acc := range p.MerchantAccountInfo {
		if tag >= "26" && tag <= "45" {
			result = append(result, acc)
		}
	}
	return result
}

// MPAN returns the Merchant PAN from the QRIS national entry (tag 51)
// if available, falling back to the first PSP entry that has one.
// Returns empty string if no MPAN is found.
func (p *Payload) MPAN() string {
	if acc, ok := p.QRISMerchantAccount(); ok && acc.MPAN != "" {
		return acc.MPAN
	}
	for _, acc := range p.PSPMerchantAccounts() {
		if acc.MPAN != "" {
			return acc.MPAN
		}
	}
	return ""
}

// MID returns the Merchant ID using the same precedence as MPAN.
func (p *Payload) MID() string {
	if acc, ok := p.QRISMerchantAccount(); ok && acc.MID != "" {
		return acc.MID
	}
	for _, acc := range p.PSPMerchantAccounts() {
		if acc.MID != "" {
			return acc.MID
		}
	}
	return ""
}
