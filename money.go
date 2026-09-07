package decimal

import "fmt"

// minorDigits gives the number of minor-unit (fractional) digits defined by
// ISO 4217 for common currencies. Currencies not listed default to 2, the
// most common case; callers with exotic currency needs should rescale
// explicitly rather than rely on this table.
var minorDigits = map[string]int32{
	"JPY": 0, "KRW": 0, "VND": 0, "CLP": 0, "ISK": 0,
	"BHD": 3, "KWD": 3, "OMR": 3, "TND": 3, "JOD": 3,
}

// MinorDigitsFor returns the standard number of minor-unit digits for an
// ISO 4217 currency code (e.g. 2 for "USD", 0 for "JPY", 3 for "BHD").
// Unrecognized codes return 2.
func MinorDigitsFor(currency string) int32 {
	if d, ok := minorDigits[currency]; ok {
		return d
	}
	return 2
}

// Money pairs a Decimal amount with an ISO 4217 currency code, so that
// mixing currencies is a compile-time-shaped but always runtime-checked
// error rather than a silently wrong number. Amount is always kept rescaled
// to the currency's standard minor-unit digits.
type Money struct {
	Amount   Decimal
	Currency string
}

// NewMoney builds a Money value, rounding amount to the currency's standard
// minor-unit digits using mode.
func NewMoney(amount Decimal, currency string, mode RoundingMode) (Money, error) {
	rescaled, err := amount.Rescale(MinorDigitsFor(currency), mode)
	if err != nil {
		return Money{}, err
	}
	return Money{Amount: rescaled, Currency: currency}, nil
}

// ParseMoney parses amount as a Decimal and builds a Money in one step.
func ParseMoney(amount, currency string, mode RoundingMode) (Money, error) {
	d, err := Parse(amount)
	if err != nil {
		return Money{}, err
	}
	return NewMoney(d, currency, mode)
}

// Add returns m + n. It fails with ErrCurrencyMismatch if the currencies
// differ — Money deliberately does not attempt currency conversion.
func (m Money) Add(n Money) (Money, error) {
	if m.Currency != n.Currency {
		return Money{}, ErrCurrencyMismatch
	}
	return Money{Amount: m.Amount.Add(n.Amount), Currency: m.Currency}, nil
}

// Sub returns m - n, subject to the same currency-matching rule as Add.
func (m Money) Sub(n Money) (Money, error) {
	if m.Currency != n.Currency {
		return Money{}, ErrCurrencyMismatch
	}
	return Money{Amount: m.Amount.Sub(n.Amount), Currency: m.Currency}, nil
}

// Mul returns m scaled by the plain (currency-less) factor qty, e.g. a unit
// price times a line-item quantity. The result keeps m's currency and is not
// re-rescaled to the currency's minor-unit digits — call NewMoney if you need
// that.
func (m Money) Mul(qty Decimal) Money {
	return Money{Amount: m.Amount.Mul(qty), Currency: m.Currency}
}

// Cmp compares m and n numerically, failing with ErrCurrencyMismatch if the
// currencies differ rather than silently comparing amounts in different
// currencies.
func (m Money) Cmp(n Money) (int, error) {
	if m.Currency != n.Currency {
		return 0, ErrCurrencyMismatch
	}
	return m.Amount.Cmp(n.Amount), nil
}

// Neg returns m with its amount negated, keeping the same currency. Useful
// for reversals, refunds, and credit notes.
func (m Money) Neg() Money {
	return Money{Amount: m.Amount.Neg(), Currency: m.Currency}
}

// IsZero reports whether m's amount is exactly zero.
func (m Money) IsZero() bool {
	return m.Amount.IsZero()
}

// AllocateEven splits m into n shares whose amounts sum back to exactly m,
// distributing the currency's smallest minor unit (e.g. the odd cent) to the
// first shares. This is the standard "split the bill" / "split a refund
// across invoice lines" algorithm — dividing evenly first and then
// distributing leftover minor units, rather than rounding each share
// independently, is what guarantees the shares reconstruct the original
// total exactly.
func (m Money) AllocateEven(n int) ([]Money, error) {
	if n <= 0 {
		return nil, fmt.Errorf("decimal: AllocateEven requires n > 0, got %d", n)
	}
	digits := MinorDigitsFor(m.Currency)
	totalMinor, err := m.Amount.MinorUnits(digits, RoundHalfEven)
	if err != nil {
		return nil, err
	}

	base := totalMinor / int64(n)
	remainder := totalMinor % int64(n)
	if remainder < 0 {
		remainder = -remainder
	}

	shares := make([]Money, n)
	for i := 0; i < n; i++ {
		units := base
		if int64(i) < remainder {
			if totalMinor < 0 {
				units--
			} else {
				units++
			}
		}
		shares[i] = Money{Amount: NewFromMinorUnits(units, digits), Currency: m.Currency}
	}
	return shares, nil
}

// String renders m as "amount CUR", e.g. "10.50 USD".
func (m Money) String() string {
	return m.Amount.String() + " " + m.Currency
}
