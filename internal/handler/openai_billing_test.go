package handler

import "testing"

func TestComputeCost(t *testing.T) {
	tests := []struct {
		name         string
		inputPrice   string
		outputPrice  string
		cachedPrice  string
		inputTokens  int
		outputTokens int
		cachedTokens int
		want         string
	}{
		{"zero tokens", "0.15000000", "0.60000000", "", 0, 0, 0, "0.000000"},
		{"input only", "0.15000000", "0.60000000", "", 1000, 0, 0, "0.000150"},
		{"input and output", "0.15000000", "0.60000000", "", 1000, 500, 0, "0.000450"},
		{"with cached", "0.15000000", "0.60000000", "0.07500000", 1000, 500, 200, "0.000435"},
		{"cached exceeds prompt", "0.15000000", "0.60000000", "0.07500000", 100, 0, 200, "0.000015"},
		{"empty prices", "", "", "", 1000, 500, 0, "0.000000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := computeCost(tt.inputPrice, tt.outputPrice, tt.cachedPrice, tt.inputTokens, tt.outputTokens, tt.cachedTokens)
			if err != nil {
				t.Fatalf("computeCost: %v", err)
			}
			if got != tt.want {
				t.Fatalf("computeCost = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestComputeCostInvalidPrice(t *testing.T) {
	if _, err := computeCost("bad", "0.60000000", "", 1, 1, 0); err == nil {
		t.Fatal("invalid price should fail")
	}
}
