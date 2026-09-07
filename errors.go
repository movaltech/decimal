package decimal

import "errors"

var (
	// ErrDivisionByZero is returned by any division operation whose divisor is zero.
	ErrDivisionByZero = errors.New("decimal: division by zero")

	// ErrInvalidSyntax is returned by Parse when the input is not a well-formed
	// decimal literal (empty, illegal characters, more than one '.', etc).
	ErrInvalidSyntax = errors.New("decimal: invalid decimal syntax")

	// ErrExponentOutOfRange is returned by Parse when a scientific-notation
	// exponent would require expanding the coefficient beyond maxParseDigits
	// digits (e.g. "1e999999999"), which is rejected instead of attempting
	// the underlying allocation.
	ErrExponentOutOfRange = errors.New("decimal: exponent out of range")

	// ErrRoundingNecessary is returned when RoundUnnecessary is the active
	// rounding mode but the exact result cannot be represented at the
	// requested scale/precision. It forces the caller to notice precision
	// loss instead of silently truncating a monetary value.
	ErrRoundingNecessary = errors.New("decimal: rounding necessary but RoundUnnecessary mode is set")

	// ErrScaleOutOfRange is returned when a negative scale is supplied to an
	// API that requires scale >= 0.
	ErrScaleOutOfRange = errors.New("decimal: scale must be >= 0")

	// ErrPrecisionOutOfRange is returned when a Context has an invalid
	// (negative) precision.
	ErrPrecisionOutOfRange = errors.New("decimal: precision must be >= 0")

	// ErrInexact is returned by QuoExact when the division does not terminate
	// within MaxExactScale digits (e.g. 1/3).
	ErrInexact = errors.New("decimal: division is not exact within the maximum supported scale")

	// ErrCurrencyMismatch is returned by Money operations that combine two
	// amounts denominated in different currencies.
	ErrCurrencyMismatch = errors.New("decimal: currency mismatch")
)
