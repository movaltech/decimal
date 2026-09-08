// Package decimal implements exact, arbitrary-precision fixed-point decimal
// arithmetic for Go, aimed squarely at money and other values where
// float64's binary rounding is unacceptable.
//
// A Decimal is represented as coeff * 10^-scale, where coeff is an
// arbitrary-precision signed integer (*big.Int) and scale is the number of
// digits to the right of the decimal point. float64 is never used internally
// by any arithmetic path, so values such as 0.1 + 0.2 are exact.
//
// Decimal is immutable: every method returns a new value and never mutates
// the receiver or its arguments, so a Decimal is safe to copy, compare with
// Equal/Cmp, and share across goroutines.
package decimal

import (
	"fmt"
	"math/big"
	"strings"
)

// Decimal is a fixed-point decimal number: coeff * 10^-scale.
//
// The zero value Decimal{} is a valid representation of 0 and is ready to
// use without further initialization.
//
// Decimal must not be compared with == or used as a map/set key: it holds
// an internal *big.Int, so two independently-constructed values that are
// numerically equal (e.g. two calls to MustParse("1.50")) are almost never
// == to each other, since struct equality compares that pointer, not the
// referenced digits. Always use Equal or Cmp for numeric comparison; if you
// need a comparable key, use d.String() as the key instead.
type Decimal struct {
	coeff *big.Int // unscaled value, including sign; nil means 0
	scale int32    // number of digits after the decimal point; always >= 0
}

// coefficient returns the unscaled integer value, treating the nil
// coefficient of the zero value as 0 without allocating when possible.
func (d Decimal) coefficient() *big.Int {
	if d.coeff == nil {
		return bigZero
	}
	return d.coeff
}

var bigZero = big.NewInt(0)
var bigTen = big.NewInt(10)

// powersOf10 caches small powers of ten; exp10 falls back to big.Int.Exp for
// anything larger.
var powersOf10 = [...]*big.Int{
	big.NewInt(1), big.NewInt(10), big.NewInt(100), big.NewInt(1000),
	big.NewInt(10000), big.NewInt(100000), big.NewInt(1000000),
	big.NewInt(10000000), big.NewInt(100000000), big.NewInt(1000000000),
}

// maxParseDigits bounds the total number of decimal digits Parse will
// materialize when expanding an exponent into trailing zeros (e.g. "1e50").
// Without this bound, a short string like "1e999999999" could force an
// allocation and digit-by-digit scan of hundreds of megabytes — a
// denial-of-service vector given that Parse is the documented entry point
// for untrusted external text (user input, JSON, CSV).
const maxParseDigits = 1_000_000

func exp10(n int32) *big.Int {
	if n >= 0 && int(n) < len(powersOf10) {
		return powersOf10[n]
	}
	return new(big.Int).Exp(bigTen, big.NewInt(int64(n)), nil)
}

// Zero is the additive identity, 0 at scale 0.
func Zero() Decimal { return Decimal{} }

// New returns unscaled * 10^-scale, e.g. New(1050, 2) is the exact value
// 10.50. scale must be >= 0.
func New(unscaled int64, scale int32) Decimal {
	return Decimal{coeff: big.NewInt(unscaled), scale: normScale(scale)}
}

// NewFromBigInt returns coeff * 10^-scale, taking ownership of a copy of
// coeff (the caller's big.Int is never mutated or retained).
func NewFromBigInt(coeff *big.Int, scale int32) Decimal {
	return Decimal{coeff: new(big.Int).Set(coeff), scale: normScale(scale)}
}

// NewFromInt64 returns the exact integer value i, at scale 0.
func NewFromInt64(i int64) Decimal { return New(i, 0) }

// normScale clamps a negative scale to 0 — there is no meaningful positive
// value a negative scale could represent, so nothing is lost by discarding
// it — and panics if scale exceeds maxParseDigits. New, NewFromBigInt, and
// NewFromMinorUnits have no error return, so an oversized scale can't be
// reported the way Rescale/QuoScale report ErrScaleOutOfRange; left
// unchecked, it would silently construct a value that later blows up as a
// multi-gigabyte allocation the first time it's formatted or rescaled
// (e.g. New(1, 2_000_000_000).String() would try to allocate a two-billion
// character string). A scale that large is never a legitimate value — it is
// always a caller bug — so this fails immediately and loudly instead.
func normScale(scale int32) int32 {
	if scale < 0 {
		return 0
	}
	if scale > maxParseDigits {
		panic(fmt.Sprintf("decimal: scale %d exceeds maximum supported scale %d", scale, maxParseDigits))
	}
	return scale
}

// Parse converts a plain decimal literal such as "-12.340" or "3.1e4" into a
// Decimal, exactly: the string is parsed digit by digit into a big.Int
// coefficient and an integer scale, and float64 is never involved. This is
// the only correct way to get external decimal text (user input, JSON,
// CSV, a database column) into a Decimal — never round-trip through
// strconv.ParseFloat.
func Parse(s string) (Decimal, error) {
	if s == "" {
		return Decimal{}, ErrInvalidSyntax
	}

	neg := false
	switch s[0] {
	case '+':
		s = s[1:]
	case '-':
		neg = true
		s = s[1:]
	}
	if s == "" {
		return Decimal{}, ErrInvalidSyntax
	}

	mantissa := s
	exp := int64(0)
	if i := strings.IndexAny(s, "eE"); i >= 0 {
		mantissa = s[:i]
		expStr := s[i+1:]
		if expStr == "" {
			return Decimal{}, ErrInvalidSyntax
		}
		e, err := parseSignedInt(expStr)
		if err != nil {
			return Decimal{}, ErrInvalidSyntax
		}
		exp = e
	}

	intPart := mantissa
	fracPart := ""
	if i := strings.IndexByte(mantissa, '.'); i >= 0 {
		intPart = mantissa[:i]
		fracPart = mantissa[i+1:]
		if strings.IndexByte(fracPart, '.') >= 0 {
			return Decimal{}, ErrInvalidSyntax
		}
	}
	if intPart == "" && fracPart == "" {
		return Decimal{}, ErrInvalidSyntax
	}
	if !isDigits(intPart) || !isDigits(fracPart) {
		return Decimal{}, ErrInvalidSyntax
	}

	// digits is never empty here: the check above already rejects the only
	// case (intPart and fracPart both empty) that could produce one.
	digits := intPart + fracPart
	scale := int64(len(fracPart)) - exp
	if scale < 0 {
		// Negative effective scale (e.g. "1e3") means trailing zeros:
		// pad the coefficient instead, keeping scale at 0. Bounded so a
		// tiny input like "1e999999999" cannot force a multi-gigabyte
		// allocation (and the CPU cost of parsing it) — see ErrExponentOutOfRange.
		padLen := -scale
		if padLen > maxParseDigits || int64(len(digits))+padLen > maxParseDigits {
			return Decimal{}, ErrExponentOutOfRange
		}
		digits += strings.Repeat("0", int(padLen))
		scale = 0
	}
	// Bounds a large positive scale (e.g. from "1e-999999999") the same way
	// the padding branch above bounds a large negative one: an unbounded
	// scale here doesn't allocate anything yet, but it silently produces a
	// Decimal whose later String()/MarshalJSON call must materialize a
	// scale-sized string (gigabytes, for a dozen input bytes).
	if scale > maxParseDigits {
		return Decimal{}, ErrExponentOutOfRange
	}

	// digits is non-empty and, by construction, made up only of the ASCII
	// digits isDigits validated above plus '0' padding, so SetString cannot
	// fail here.
	coeff, _ := new(big.Int).SetString(digits, 10)
	if neg && coeff.Sign() != 0 {
		coeff.Neg(coeff)
	}
	return Decimal{coeff: coeff, scale: int32(scale)}, nil
}

// MustParse is like Parse but panics on error. Intended for tests and
// package-level literals where the input is a compile-time constant known
// to be valid, e.g. var fee = decimal.MustParse("2.50").
func MustParse(s string) Decimal {
	d, err := Parse(s)
	if err != nil {
		panic("decimal: MustParse(" + s + "): " + err.Error())
	}
	return d
}

func isDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// parseSignedInt is only ever called by Parse with a non-empty exponent
// string, but the length check is folded into the sign test below (rather
// than a separate leading guard) so indexing s[0] can never go out of range
// even if that guarantee ever changes.
func parseSignedInt(s string) (int64, error) {
	neg := false
	if len(s) > 0 && (s[0] == '+' || s[0] == '-') {
		neg = s[0] == '-'
		s = s[1:]
	}
	if s == "" || !isDigits(s) {
		return 0, ErrInvalidSyntax
	}
	var v int64
	for _, c := range s {
		v = v*10 + int64(c-'0')
	}
	if neg {
		v = -v
	}
	return v, nil
}

// Scale returns the number of digits to the right of the decimal point.
func (d Decimal) Scale() int32 { return d.scale }

// Sign returns -1, 0, or 1 depending on the sign of d.
func (d Decimal) Sign() int { return d.coefficient().Sign() }

// IsZero reports whether d is exactly zero (at any scale).
func (d Decimal) IsZero() bool { return d.coefficient().Sign() == 0 }

// Neg returns -d.
func (d Decimal) Neg() Decimal {
	return Decimal{coeff: new(big.Int).Neg(d.coefficient()), scale: d.scale}
}

// Abs returns |d|.
func (d Decimal) Abs() Decimal {
	return Decimal{coeff: new(big.Int).Abs(d.coefficient()), scale: d.scale}
}

// Cmp compares d and e numerically (independent of their individual scales:
// 1.50 and 1.5 compare equal) and returns -1, 0, or 1.
func (d Decimal) Cmp(e Decimal) int {
	scale := maxInt32(d.scale, e.scale)
	a := d.rescaleUpTo(scale)
	b := e.rescaleUpTo(scale)
	return a.coefficient().Cmp(b.coefficient())
}

// Equal reports whether d and e are numerically equal, regardless of scale.
func (d Decimal) Equal(e Decimal) bool { return d.Cmp(e) == 0 }

// GreaterThan reports whether d > e.
func (d Decimal) GreaterThan(e Decimal) bool { return d.Cmp(e) > 0 }

// GreaterThanOrEqual reports whether d >= e.
func (d Decimal) GreaterThanOrEqual(e Decimal) bool { return d.Cmp(e) >= 0 }

// LessThan reports whether d < e.
func (d Decimal) LessThan(e Decimal) bool { return d.Cmp(e) < 0 }

// LessThanOrEqual reports whether d <= e.
func (d Decimal) LessThanOrEqual(e Decimal) bool { return d.Cmp(e) <= 0 }

// IsPositive reports whether d > 0.
func (d Decimal) IsPositive() bool { return d.Sign() > 0 }

// IsNegative reports whether d < 0.
func (d Decimal) IsNegative() bool { return d.Sign() < 0 }

func maxInt32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

// rescaleUpTo multiplies the coefficient so the scale becomes newScale. It
// is only valid when newScale >= d.scale, in which case the operation is
// always exact (no rounding, ever).
func (d Decimal) rescaleUpTo(newScale int32) Decimal {
	if newScale <= d.scale {
		return d
	}
	factor := exp10(newScale - d.scale)
	return Decimal{coeff: new(big.Int).Mul(d.coefficient(), factor), scale: newScale}
}

// String renders d in plain fixed-point notation, e.g. "-12.340". It never
// uses scientific notation, so it round-trips through Parse exactly and is
// safe to store as text (JSON, CSV, a VARCHAR column) without any loss.
func (d Decimal) String() string {
	coeff := d.coefficient()
	neg := coeff.Sign() < 0
	digits := new(big.Int).Abs(coeff).String()

	scale := int(d.scale)
	if scale == 0 {
		if neg {
			return "-" + digits
		}
		return digits
	}

	if len(digits) <= scale {
		digits = strings.Repeat("0", scale-len(digits)+1) + digits
	}
	intPart := digits[:len(digits)-scale]
	fracPart := digits[len(digits)-scale:]

	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	b.WriteString(intPart)
	b.WriteByte('.')
	b.WriteString(fracPart)
	return b.String()
}

// numDigits returns the number of decimal digits in |coeff|, treating 0 as
// having 1 digit.
func numDigits(coeff *big.Int) int {
	if coeff.Sign() == 0 {
		return 1
	}
	// big.Int has no direct digit-count API cheaper than formatting for
	// arbitrary size; this is only called on rounding paths, not hot loops.
	abs := coeff
	if coeff.Sign() < 0 {
		abs = new(big.Int).Abs(coeff)
	}
	return len(abs.Text(10))
}
