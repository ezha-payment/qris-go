package qris

// This file defines typed constants for the enum-like QRIS fields
// (merchant criteria, currency, country, and globally unique
// identifier). They exist to make call sites self-documenting and to
// let editors offer completion, while remaining 100% compatible with
// the string-based Builder API.
//
// Design choice: the Builder setters keep their string parameters
// rather than gaining typed overloads. Because every type below has an
// underlying type of string, a typed constant is passed with an
// explicit conversion:
//
//	b.Currency(string(qris.CurrencyIDR))
//	b.AddPSPAccount("26", string(qris.GUIDana), mpan, mid, string(qris.CriteriaUMI))
//
// This keeps a single setter per field (no API surface growth, no
// behavioural change for existing string callers) at the cost of one
// explicit conversion at the call site.
//
// Deliberately, none of these types override String(): the values are
// embedded verbatim into payloads and appear in logs, so the default
// fmt representation must remain the raw code (e.g. "360", not
// "Indonesian Rupiah"). Human-readable names are exposed via the
// Description method instead.

// MerchantCriteria is the merchant size classification carried in
// sub-tag 03 of a merchant account block.
type MerchantCriteria string

const (
	// CriteriaUMI is Usaha Mikro (micro enterprise).
	CriteriaUMI MerchantCriteria = "UMI"
	// CriteriaUKE is Usaha Kecil (small enterprise).
	CriteriaUKE MerchantCriteria = "UKE"
	// CriteriaUME is Usaha Menengah (medium enterprise).
	CriteriaUME MerchantCriteria = "UME"
	// CriteriaUBE is Usaha Besar (large enterprise).
	CriteriaUBE MerchantCriteria = "UBE"
	// CriteriaURE is Regular (non-classified).
	CriteriaURE MerchantCriteria = "URE"
)

// Description returns the human-readable name of the merchant criteria,
// or "Unknown" if the value is not recognized.
func (c MerchantCriteria) Description() string {
	switch c {
	case CriteriaUMI:
		return "Micro"
	case CriteriaUKE:
		return "Small"
	case CriteriaUME:
		return "Medium"
	case CriteriaUBE:
		return "Large"
	case CriteriaURE:
		return "Regular"
	default:
		return "Unknown"
	}
}

// CurrencyCode is an ISO 4217 numeric currency code carried in tag 53.
type CurrencyCode string

const (
	// CurrencyIDR is Indonesian Rupiah, the QRIS default.
	CurrencyIDR CurrencyCode = "360"
	// CurrencyUSD is United States Dollar.
	CurrencyUSD CurrencyCode = "840"
	// CurrencySGD is Singapore Dollar.
	CurrencySGD CurrencyCode = "702"
	// CurrencyMYR is Malaysian Ringgit.
	CurrencyMYR CurrencyCode = "458"
	// CurrencyTHB is Thai Baht.
	CurrencyTHB CurrencyCode = "764"
	// CurrencyJPY is Japanese Yen.
	CurrencyJPY CurrencyCode = "392"
	// CurrencyKRW is South Korean Won.
	CurrencyKRW CurrencyCode = "410"
)

// Description returns the human-readable name of the currency, or
// "Unknown" if the value is not recognized.
func (c CurrencyCode) Description() string {
	switch c {
	case CurrencyIDR:
		return "Indonesian Rupiah"
	case CurrencyUSD:
		return "US Dollar"
	case CurrencySGD:
		return "Singapore Dollar"
	case CurrencyMYR:
		return "Malaysian Ringgit"
	case CurrencyTHB:
		return "Thai Baht"
	case CurrencyJPY:
		return "Japanese Yen"
	case CurrencyKRW:
		return "South Korean Won"
	default:
		return "Unknown"
	}
}

// CountryCode is an ISO 3166-1 alpha-2 country code carried in tag 58.
type CountryCode string

const (
	// CountryID is Indonesia, the QRIS default.
	CountryID CountryCode = "ID"
	// CountrySG is Singapore.
	CountrySG CountryCode = "SG"
	// CountryMY is Malaysia.
	CountryMY CountryCode = "MY"
	// CountryTH is Thailand.
	CountryTH CountryCode = "TH"
	// CountryJP is Japan.
	CountryJP CountryCode = "JP"
	// CountryKR is South Korea.
	CountryKR CountryCode = "KR"
)

// Description returns the human-readable name of the country, or
// "Unknown" if the value is not recognized.
func (c CountryCode) Description() string {
	switch c {
	case CountryID:
		return "Indonesia"
	case CountrySG:
		return "Singapore"
	case CountryMY:
		return "Malaysia"
	case CountryTH:
		return "Thailand"
	case CountryJP:
		return "Japan"
	case CountryKR:
		return "South Korea"
	default:
		return "Unknown"
	}
}

// GUI is a Globally Unique Identifier carried in sub-tag 00 of a
// merchant account block. It identifies the routing network (national
// QRIS) or the specific Payment Service Provider.
type GUI string

const (
	// GUIQRISNational is the national QRIS switching identifier, used in
	// tag 51.
	GUIQRISNational GUI = "ID.CO.QRIS.WWW"
	// GUIDana is the DANA wallet identifier.
	GUIDana GUI = "ID.DANA.WWW"
	// GUIShopeePay is the ShopeePay identifier.
	GUIShopeePay GUI = "ID.SHOPEEPAY.WWW"
	// GUIOVO is the OVO identifier.
	GUIOVO GUI = "ID.OVO.WWW"
	// GUILinkAja is the LinkAja identifier.
	GUILinkAja GUI = "ID.LINKAJA.WWW"
	// GUIGoPay is the GoPay (Gojek) identifier.
	GUIGoPay GUI = "ID.CO.GOJEK.WWW"

	// GUIPayNowSG is Singapore's PayNow identifier, carried as the
	// reverse-domain GUI in sub-tag 00 of tag 26 on SGQR payloads. It is
	// relevant to QRIS cross-border acceptance with Singapore.
	GUIPayNowSG GUI = "SG.PAYNOW"

	// GUIPromptPayTH is Thailand's PromptPay identifier. Unlike the
	// reverse-domain GUIs above, PromptPay uses an EMVCo Application
	// Identifier (AID) carried in sub-tag 00 of tag 29 (or tag 30). It
	// is relevant to QRIS cross-border acceptance with Thailand.
	GUIPromptPayTH GUI = "A000000677010111"
)

// Description returns the human-readable name of the GUI's network or
// provider, or "Unknown" if the value is not recognized.
func (g GUI) Description() string {
	switch g {
	case GUIQRISNational:
		return "National QRIS"
	case GUIDana:
		return "DANA"
	case GUIShopeePay:
		return "ShopeePay"
	case GUIOVO:
		return "OVO"
	case GUILinkAja:
		return "LinkAja"
	case GUIGoPay:
		return "GoPay"
	case GUIPayNowSG:
		return "PayNow (Singapore)"
	case GUIPromptPayTH:
		return "PromptPay (Thailand)"
	default:
		return "Unknown"
	}
}
