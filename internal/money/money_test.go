package money

import "testing"

func TestParse6Valid(t *testing.T) {
	tests := []struct {
		in   string
		want Amount
	}{
		{"0", 0},
		{"1", 1_000_000},
		{"100.000000", 100_000_000},
		{"-12.500000", -12_500_000},
		{"0.000001", 1},
		{"-0.000001", -1},
		{"1.", 1_000_000},
		{"123456.789012", 123456_789012},
	}
	for _, tt := range tests {
		got, err := Parse6(tt.in)
		if err != nil {
			t.Fatalf("Parse6(%q) error: %v", tt.in, err)
		}
		if got != tt.want {
			t.Fatalf("Parse6(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestParse6Invalid(t *testing.T) {
	tests := []string{
		"",
		"abc",
		"1.2.3",
		"1.2345678",
		".5",
		"+1",
		"1e5",
		"12,5",
		"-",
		" ",
	}
	for _, in := range tests {
		if _, err := Parse6(in); err == nil {
			t.Fatalf("Parse6(%q) = nil error, want error", in)
		}
	}
}

func TestParse8Valid(t *testing.T) {
	tests := []struct {
		in   string
		want Amount
	}{
		{"0.15000000", 15_000_000},
		{"0.60000000", 60_000_000},
		{"1", 100_000_000},
		{"-0.00000001", -1},
	}
	for _, tt := range tests {
		got, err := Parse8(tt.in)
		if err != nil {
			t.Fatalf("Parse8(%q) error: %v", tt.in, err)
		}
		if got != tt.want {
			t.Fatalf("Parse8(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestParse8Invalid(t *testing.T) {
	if _, err := Parse8("0.123456789"); err == nil {
		t.Fatal("Parse8 with 9 decimals should fail")
	}
	if _, err := Parse8("x"); err == nil {
		t.Fatal("Parse8 with letters should fail")
	}
}

func TestFormat6RoundTrip(t *testing.T) {
	canonical := []string{
		"0.000000",
		"1.000000",
		"100.000000",
		"-12.500000",
		"0.000001",
		"-0.000001",
		"123456.789012",
	}
	for _, in := range canonical {
		amount, err := Parse6(in)
		if err != nil {
			t.Fatalf("Parse6(%q) error: %v", in, err)
		}
		if got := Format6(amount); got != in {
			t.Fatalf("Format6(Parse6(%q)) = %q, want %q", in, got, in)
		}
	}
}

func TestFormat8RoundTrip(t *testing.T) {
	canonical := []string{
		"0.00000000",
		"0.15000000",
		"0.60000000",
		"-0.00000001",
		"12345678.12345678",
	}
	for _, in := range canonical {
		amount, err := Parse8(in)
		if err != nil {
			t.Fatalf("Parse8(%q) error: %v", in, err)
		}
		if got := Format8(amount); got != in {
			t.Fatalf("Format8(Parse8(%q)) = %q, want %q", in, got, in)
		}
	}
}

func TestFormatPadsAndNormalizes(t *testing.T) {
	amount, err := Parse6("1")
	if err != nil {
		t.Fatal(err)
	}
	if got := Format6(amount); got != "1.000000" {
		t.Fatalf("Format6(1) = %q, want 1.000000", got)
	}

	amount, err = Parse6("-0")
	if err != nil {
		t.Fatal(err)
	}
	if got := Format6(amount); got != "0.000000" {
		t.Fatalf("Format6(-0) = %q, want 0.000000", got)
	}
}

func TestArithmeticAndCompare(t *testing.T) {
	a, _ := Parse6("10.000000")
	b, _ := Parse6("2.500000")

	if got := Format6(a.Add(b)); got != "12.500000" {
		t.Fatalf("Add = %q, want 12.500000", got)
	}
	if got := Format6(a.Sub(b)); got != "7.500000" {
		t.Fatalf("Sub = %q, want 7.500000", got)
	}
	if a.Cmp(b) != 1 {
		t.Fatal("Cmp(10, 2.5) should be 1")
	}
	if b.Cmp(a) != -1 {
		t.Fatal("Cmp(2.5, 10) should be -1")
	}
	if a.Cmp(a) != 0 {
		t.Fatal("Cmp(equal) should be 0")
	}
	if !(Amount(0)).IsZero() {
		t.Fatal("zero Amount should be zero")
	}
}

func TestParse6RejectsOverflow(t *testing.T) {
	if _, err := Parse6("99999999999999999999.000000"); err == nil {
		t.Fatal("overflowing amount should fail")
	}
}
