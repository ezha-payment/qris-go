package qris

import "testing"

// TestConstants_StringValues pins the raw code carried by every typed
// constant. These values are wire-format and must never drift.
func TestConstants_StringValues(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"CriteriaUMI", string(CriteriaUMI), "UMI"},
		{"CriteriaUKE", string(CriteriaUKE), "UKE"},
		{"CriteriaUME", string(CriteriaUME), "UME"},
		{"CriteriaUBE", string(CriteriaUBE), "UBE"},
		{"CriteriaURE", string(CriteriaURE), "URE"},

		{"CurrencyIDR", string(CurrencyIDR), "360"},
		{"CurrencyUSD", string(CurrencyUSD), "840"},
		{"CurrencySGD", string(CurrencySGD), "702"},
		{"CurrencyMYR", string(CurrencyMYR), "458"},
		{"CurrencyTHB", string(CurrencyTHB), "764"},
		{"CurrencyJPY", string(CurrencyJPY), "392"},
		{"CurrencyKRW", string(CurrencyKRW), "410"},

		{"CountryID", string(CountryID), "ID"},
		{"CountrySG", string(CountrySG), "SG"},
		{"CountryMY", string(CountryMY), "MY"},
		{"CountryTH", string(CountryTH), "TH"},
		{"CountryJP", string(CountryJP), "JP"},
		{"CountryKR", string(CountryKR), "KR"},

		{"GUIQRISNational", string(GUIQRISNational), "ID.CO.QRIS.WWW"},
		{"GUIDana", string(GUIDana), "ID.DANA.WWW"},
		{"GUIShopeePay", string(GUIShopeePay), "ID.SHOPEEPAY.WWW"},
		{"GUIOVO", string(GUIOVO), "ID.OVO.WWW"},
		{"GUILinkAja", string(GUILinkAja), "ID.LINKAJA.WWW"},
		{"GUIGoPay", string(GUIGoPay), "ID.CO.GOJEK.WWW"},

		{"GUIPayNowSG", string(GUIPayNowSG), "SG.PAYNOW"},
		{"GUIPromptPayTH", string(GUIPromptPayTH), "A000000677010111"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.name, c.got, c.want)
		}
	}
}

// TestConstants_Description spot-checks the human-readable names and the
// Unknown fallback.
func TestConstants_Description(t *testing.T) {
	if got := CurrencyIDR.Description(); got != "Indonesian Rupiah" {
		t.Errorf("CurrencyIDR.Description() = %q", got)
	}
	if got := CountryID.Description(); got != "Indonesia" {
		t.Errorf("CountryID.Description() = %q", got)
	}
	if got := CriteriaUMI.Description(); got != "Micro" {
		t.Errorf("CriteriaUMI.Description() = %q", got)
	}
	if got := GUIDana.Description(); got != "DANA" {
		t.Errorf("GUIDana.Description() = %q", got)
	}
	if got := GUIPayNowSG.Description(); got != "PayNow (Singapore)" {
		t.Errorf("GUIPayNowSG.Description() = %q", got)
	}
	if got := GUIPromptPayTH.Description(); got != "PromptPay (Thailand)" {
		t.Errorf("GUIPromptPayTH.Description() = %q", got)
	}
	if got := CurrencyCode("999").Description(); got != "Unknown" {
		t.Errorf("unknown currency Description() = %q, want Unknown", got)
	}
}

// TestConstants_BuilderAccepts proves the typed constants flow through
// the string-based Builder API via explicit conversion and round-trip.
func TestConstants_BuilderAccepts(t *testing.T) {
	payload, err := NewBuilder().
		Static().
		MerchantName("WARUNG SAMPLE").
		MerchantCity("JAKARTA").
		MCC("4812").
		Currency(string(CurrencyIDR)).
		Country(string(CountryID)).
		AddPSPAccount("26", string(GUIDana), "936008990000000001", "002100000001", string(CriteriaUMI)).
		AddQRISAccount("9360091500001234567", "ID1020012345678", string(CriteriaUKE)).
		Build()
	if err != nil {
		t.Fatalf("Build() with typed constants failed: %v", err)
	}

	p, err := Parse(payload)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}
	if p.TransactionCurrency != string(CurrencyIDR) {
		t.Errorf("currency: got %q, want %q", p.TransactionCurrency, CurrencyIDR)
	}
	if p.CountryCode != string(CountryID) {
		t.Errorf("country: got %q, want %q", p.CountryCode, CountryID)
	}
	psp := p.PSPMerchantAccounts()
	if len(psp) != 1 || psp[0].GloballyUniqueIdentifier != string(GUIDana) {
		t.Errorf("PSP GUI: got %+v, want %q", psp, GUIDana)
	}
	if acc, ok := p.QRISMerchantAccount(); !ok || acc.MerchantCriteria != string(CriteriaUKE) {
		t.Errorf("tag 51 criteria: got %+v, want %q", acc, CriteriaUKE)
	}
}

// TestConstants_ValidationAccepts confirms Validate treats the typed
// constants' string values as valid (they are strings underneath).
func TestConstants_ValidationAccepts(t *testing.T) {
	p := &Payload{
		PayloadFormatIndicator:  "01",
		PointOfInitiationMethod: "11",
		MerchantAccountInfo: map[string]MerchantAccount{
			"51": {
				GloballyUniqueIdentifier: string(GUIQRISNational),
				MPAN:                     "9360091500001234567",
				MerchantCriteria:         string(CriteriaUBE),
			},
		},
		MerchantCategoryCode: "4812",
		TransactionCurrency:  string(CurrencyIDR),
		CountryCode:          string(CountryID),
		MerchantName:         "WARUNG SAMPLE",
		MerchantCity:         "JAKARTA",
	}
	if err := Validate(p); err != nil {
		t.Fatalf("Validate rejected typed-constant values: %v", err)
	}

	// Every criteria constant must pass.
	for _, c := range []MerchantCriteria{CriteriaUMI, CriteriaUKE, CriteriaUME, CriteriaUBE, CriteriaURE} {
		acc := p.MerchantAccountInfo["51"]
		acc.MerchantCriteria = string(c)
		p.MerchantAccountInfo["51"] = acc
		if err := Validate(p); err != nil {
			t.Errorf("Validate rejected criteria %q: %v", c, err)
		}
	}
}
