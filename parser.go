package qris

import (
	"errors"
	"fmt"
	"strconv"
)

// Errors returned during parsing.
var (
	ErrPayloadTooShort  = errors.New("qris: payload too short")
	ErrInvalidTLVFormat = errors.New("qris: invalid TLV format")
	ErrInvalidCRC       = errors.New("qris: CRC validation failed")
	ErrInvalidLength    = errors.New("qris: invalid length field")
	ErrMissingCRCTag    = errors.New("qris: missing CRC tag")
)

// Parse decodes a QRIS payload string into a Payload struct.
//
// The payload must end with the CRC tag "6304" followed by a
// 4-character hexadecimal CRC value. Returns an error if the
// payload is malformed or the CRC validation fails.
func Parse(payload string) (*Payload, error) {
	if len(payload) < 8 {
		return nil, ErrPayloadTooShort
	}

	// Step 1: Locate CRC tag at position len-8.
	crcTagStart := len(payload) - 8
	if payload[crcTagStart:crcTagStart+4] != "6304" {
		return nil, fmt.Errorf("expected '6304' at position %d, got %q: %w",
			crcTagStart, payload[crcTagStart:crcTagStart+4], ErrMissingCRCTag)
	}

	// Step 2: Verify CRC over [start .. "6304") excluding CRC value.
	dataToCheck := payload[:len(payload)-4]
	expectedCRC := formatCRC(computeCRC([]byte(dataToCheck)))
	actualCRC := payload[len(payload)-4:]
	if expectedCRC != actualCRC {
		return nil, fmt.Errorf("expected %s, got %s: %w",
			expectedCRC, actualCRC, ErrInvalidCRC)
	}

	// Step 3: Tokenize TLV portion (everything before the CRC tag).
	tokens, err := tokenize(payload[:crcTagStart])
	if err != nil {
		return nil, err
	}

	// Step 4: Map tokens to Payload struct.
	p := &Payload{
		CRC:                 actualCRC,
		MerchantAccountInfo: make(map[string]MerchantAccount),
	}
	for _, t := range tokens {
		switch {
		case t.tag == "00":
			p.PayloadFormatIndicator = t.value
		case t.tag == "01":
			p.PointOfInitiationMethod = t.value
		case isMerchantAccountTag(t.tag):
			// Tags 26-51 contain nested TLV.
			acc, err := parseMerchantAccount(t.value)
			if err != nil {
				return nil, fmt.Errorf("parsing merchant account tag %s: %w", t.tag, err)
			}
			p.MerchantAccountInfo[t.tag] = acc
		case t.tag == "52":
			p.MerchantCategoryCode = t.value
		case t.tag == "53":
			p.TransactionCurrency = t.value
		case t.tag == "54":
			p.TransactionAmount = t.value
		case t.tag == "58":
			p.CountryCode = t.value
		case t.tag == "59":
			p.MerchantName = t.value
		case t.tag == "60":
			p.MerchantCity = t.value
		case t.tag == "61":
			p.PostalCode = t.value
		case t.tag == "62":
			ad, err := parseAdditionalData(t.value)
			if err != nil {
				return nil, fmt.Errorf("parsing additional data tag 62: %w", err)
			}
			p.AdditionalData = ad
		}
		// Unknown tags are silently ignored for forward-compat.
	}

	// Clean up empty map for nicer zero-value semantics.
	if len(p.MerchantAccountInfo) == 0 {
		p.MerchantAccountInfo = nil
	}

	return p, nil
}

// isMerchantAccountTag reports whether a tag falls in the
// Merchant Account Information range (26 through 51 inclusive).
func isMerchantAccountTag(tag string) bool {
	n, err := strconv.Atoi(tag)
	if err != nil {
		return false
	}
	return n >= 26 && n <= 51
}

// parseMerchantAccount parses the nested TLV inside a tag 26-51 value.
func parseMerchantAccount(value string) (MerchantAccount, error) {
	subTokens, err := tokenize(value)
	if err != nil {
		return MerchantAccount{}, err
	}
	acc := MerchantAccount{Raw: make(map[string]string)}
	for _, st := range subTokens {
		switch st.tag {
		case "00":
			acc.GloballyUniqueIdentifier = st.value
		case "01":
			acc.MPAN = st.value
		case "02":
			acc.MID = st.value
		case "03":
			acc.MerchantCriteria = st.value
		default:
			acc.Raw[st.tag] = st.value
		}
	}
	if len(acc.Raw) == 0 {
		acc.Raw = nil
	}
	return acc, nil
}

// parseAdditionalData parses the nested TLV inside tag 62.
func parseAdditionalData(value string) (*AdditionalData, error) {
	subTokens, err := tokenize(value)
	if err != nil {
		return nil, err
	}
	ad := &AdditionalData{Raw: make(map[string]string)}
	for _, st := range subTokens {
		switch st.tag {
		case "01":
			ad.BillNumber = st.value
		case "02":
			ad.MobileNumber = st.value
		case "03":
			ad.StoreLabel = st.value
		case "04":
			ad.LoyaltyNumber = st.value
		case "05":
			ad.ReferenceLabel = st.value
		case "06":
			ad.CustomerLabel = st.value
		case "07":
			ad.TerminalLabel = st.value
		case "08":
			ad.PurposeOfTransaction = st.value
		default:
			ad.Raw[st.tag] = st.value
		}
	}
	if len(ad.Raw) == 0 {
		ad.Raw = nil
	}
	return ad, nil
}

// token represents one Tag-Length-Value triplet in a TLV-encoded string.
type token struct {
	tag    string
	length int
	value  string
}

// tokenize splits a TLV-encoded string into individual tokens.
//
// QRIS uses ASCII-encoded TLV: each token has a 2-character tag,
// a 2-character length (zero-padded decimal), and a value of the
// specified length.
func tokenize(s string) ([]token, error) {
	var tokens []token
	i := 0
	for i < len(s) {
		if i+4 > len(s) {
			return nil, fmt.Errorf("incomplete TLV header at position %d: %w",
				i, ErrInvalidTLVFormat)
		}

		tag := s[i : i+2]
		lengthStr := s[i+2 : i+4]

		length, err := strconv.Atoi(lengthStr)
		if err != nil {
			return nil, fmt.Errorf("invalid length %q for tag %s: %w",
				lengthStr, tag, ErrInvalidLength)
		}

		if i+4+length > len(s) {
			return nil, fmt.Errorf("value of length %d for tag %s exceeds remaining payload: %w",
				length, tag, ErrInvalidTLVFormat)
		}

		value := s[i+4 : i+4+length]
		tokens = append(tokens, token{tag: tag, length: length, value: value})
		i += 4 + length
	}
	return tokens, nil
}
