# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [0.1.1] - 2026-09-07

### Added
- `Money.Mul(qty Decimal) Money` — scales a Money amount by a plain quantity,
  e.g. a unit price times a line-item quantity.
- `Money.Cmp(n Money) (int, error)` — compares two Money values, returning
  `ErrCurrencyMismatch` if the currencies differ.
- `Money.Neg() Money` — negates a Money amount, keeping its currency.
  Useful for reversals, refunds, and credit notes.
- `Money.IsZero() bool` — reports whether a Money amount is exactly zero.
- `Decimal.Pow(n uint) (Decimal, error)` — exact integer exponentiation,
  computed by repeated squaring on the underlying coefficient. Enables
  compound-interest and annuity calculations (e.g. `(1 + rate).Pow(periods)`)
  without accumulating rounding error across periods. Fails with
  `ErrExponentOutOfRange` if the resulting scale would be unreasonably large.
- Benchmarks for `Add`, `Mul`, `QuoScale`, `Parse`, `String`, and `Pow`.
- Runnable examples for `Parse`, `Decimal.Add`, `Decimal.Pow`,
  `Money.AllocateEven`, and `Money.Mul`.

## [0.1.0] - 2026-09-06

### Added
- Immutable `Decimal` type (`coeff *big.Int` + `scale int32`), with the zero
  value representing `0` and ready to use without initialization.
- Constructors: `Zero`, `New`, `NewFromBigInt`, `NewFromInt64`,
  `Parse`/`MustParse`, `NewFromMinorUnits`.
- `Parse`, hardened against pathological input: bounds exponent expansion
  (`maxParseDigits`) so a literal like `"1e999999999"` is rejected instead of
  attempting the underlying allocation.
- Accessors and predicates: `Scale`, `Sign`, `IsZero`, `IsPositive`,
  `IsNegative`.
- `Neg`, `Abs`.
- Scale-independent comparison: `Cmp`, `Equal`, `GreaterThan(OrEqual)`,
  `LessThan(OrEqual)`.
- Exact arithmetic: `Add`, `Sub`, `Mul`.
- `Rescale(newScale, mode)` with eight rounding modes: `RoundHalfEven`,
  `RoundHalfUp`, `RoundHalfDown`, `RoundUp`, `RoundDown`, `RoundCeiling`,
  `RoundFloor`, `RoundUnnecessary` (fails instead of rounding silently).
- Three explicit division paths: `QuoScale` (fixed decimal places, for
  money), `QuoContext` (significant digits via `Context`/`DefaultContext`),
  `QuoExact` (fails with `ErrInexact` if the quotient doesn't terminate
  within `MaxExactScale`).
- Conversions: `MinorUnits`, `Int64`, `BigInt`, `Float64`.
- `Money`: `NewMoney`, `ParseMoney`, `Add`, `Sub`, `AllocateEven`, `String`.
- `MinorDigitsFor`, backed by an ISO 4217 minor-unit digits table.
- `float64`-free serialization: `MarshalJSON`/`UnmarshalJSON` (as a quoted
  string), `MarshalText`/`UnmarshalText`, `driver.Valuer`/`sql.Scanner`.
- Sentinel errors: `ErrDivisionByZero`, `ErrInvalidSyntax`,
  `ErrExponentOutOfRange`, `ErrRoundingNecessary`, `ErrScaleOutOfRange`,
  `ErrPrecisionOutOfRange`, `ErrInexact`, `ErrCurrencyMismatch`.
- Full unit test suite, a property-based test checked against
  `math/big.Rat`, and fuzz targets for `Parse` and `Rescale`.

[0.1.1]: https://github.com/diegofxm/decimal/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/diegofxm/decimal/releases/tag/v0.1.0
