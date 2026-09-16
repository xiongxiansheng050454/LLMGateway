// Package money provides fixed-point decimal amounts backed by int64 scaled
// units. It never uses float64, so amounts can be compared, added and formatted
// without rounding surprises.
//
// An Amount stores the value in the scale it was parsed with (6 decimals via
// Parse6, 8 decimals via Parse8). Do not mix Amount values produced by Parse6
// and Parse8; add/subtract/compare only values of the same scale.
package money

import (
	"errors"
	"strconv"
	"strings"
)

// ErrInvalid is returned for any malformed amount string.
var ErrInvalid = errors.New("invalid amount")

// Amount is a fixed-point decimal amount in scaled integer units.
type Amount int64

// Parse6 parses a decimal string into an Amount scaled by 1e-6 (balances).
func Parse6(value string) (Amount, error) {
	return parse(value, 6)
}

// Parse8 parses a decimal string into an Amount scaled by 1e-8 (unit prices).
func Parse8(value string) (Amount, error) {
	return parse(value, 8)
}

// Format6 renders an Amount as a string with exactly 6 decimal places.
func Format6(amount Amount) string {
	return format(amount, 6)
}

// Format8 renders an Amount as a string with exactly 8 decimal places.
func Format8(amount Amount) string {
	return format(amount, 8)
}

// Add returns a + b. Both operands must share the same scale.
func (a Amount) Add(b Amount) Amount {
	return a + b
}

// Sub returns a - b. Both operands must share the same scale.
func (a Amount) Sub(b Amount) Amount {
	return a - b
}

// Cmp compares two same-scale amounts, returning -1, 0 or 1.
func (a Amount) Cmp(b Amount) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// IsZero reports whether the amount is exactly zero.
func (a Amount) IsZero() bool {
	return a == 0
}

func parse(value string, scale int) (Amount, error) {
	if value == "" {
		return 0, ErrInvalid
	}

	negative := strings.HasPrefix(value, "-")
	if negative {
		value = value[1:]
	}
	if value == "" {
		return 0, ErrInvalid
	}

	intPart, fracPart, hasFraction := strings.Cut(value, ".")
	if !isDigits(intPart) {
		return 0, ErrInvalid
	}
	if hasFraction {
		if len(fracPart) > scale || !isDigitsOrEmpty(fracPart) {
			return 0, ErrInvalid
		}
	}

	for len(fracPart) < scale {
		fracPart += "0"
	}

	scaled, err := strconv.ParseInt(intPart+fracPart, 10, 64)
	if err != nil {
		return 0, ErrInvalid
	}
	if negative {
		scaled = -scaled
	}
	return Amount(scaled), nil
}

func format(amount Amount, scale int) string {
	digits := strconv.FormatInt(int64(amount), 10)
	negative := strings.HasPrefix(digits, "-")
	if negative {
		digits = digits[1:]
	}
	for len(digits) <= scale {
		digits = "0" + digits
	}

	result := digits[:len(digits)-scale] + "." + digits[len(digits)-scale:]
	if negative {
		result = "-" + result
	}
	return result
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}
	return isDigitsOrEmpty(value)
}

func isDigitsOrEmpty(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return true
}
