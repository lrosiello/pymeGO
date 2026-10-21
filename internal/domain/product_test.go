package domain

import "testing"

func floatPtr(value float64) *float64 {
	return &value
}

func TestParsePriceMode(t *testing.T) {
	cases := []struct {
		raw     string
		want    PriceMode
		wantErr bool
	}{
		{raw: "", want: PriceModeB2C},
		{raw: "b2c", want: PriceModeB2C},
		{raw: " B2C ", want: PriceModeB2C},
		{raw: "b2b", want: PriceModeB2B},
		{raw: "B2B", want: PriceModeB2B},
		{raw: "mayorista", wantErr: true},
	}

	for _, tc := range cases {
		got, err := ParsePriceMode(tc.raw)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParsePriceMode(%q) = %q, se esperaba error", tc.raw, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParsePriceMode(%q) error inesperado: %v", tc.raw, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParsePriceMode(%q) = %q, se esperaba %q", tc.raw, got, tc.want)
		}
	}
}

func TestPriceForMode(t *testing.T) {
	variant := &ProductVariant{Price: 1500, B2BPrice: floatPtr(1200)}

	if got := variant.PriceForMode(PriceModeB2C); got != 1500 {
		t.Errorf("PriceForMode(B2C) = %v, se esperaba 1500", got)
	}
	if got := variant.PriceForMode(PriceModeB2B); got != 1200 {
		t.Errorf("PriceForMode(B2B) = %v, se esperaba 1200", got)
	}

	variant.B2BPrice = nil
	if got := variant.PriceForMode(PriceModeB2B); got != 1500 {
		t.Errorf("PriceForMode(B2B sin b2b_price) = %v, se esperaba fallback 1500", got)
	}
}
