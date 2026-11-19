# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [1.0.1] - 2026-11-19

### Fixed
- `Parse` no longer mis-parses or panics on scientific-notation exponents
  beyond the int64 range. The exponent accumulator used to wrap around
  silently, so `"1e18446744073709551616"` parsed as `1`, and
  `"1e-9223372036854775808"` panicked with a negative `strings.Repeat`
  count. Any exponent above 2^40 in magnitude now fails with
  `ErrExponentOutOfRange`, which is reachable from untrusted input
  (JSON, `Scan`, `UnmarshalText`).
- `ErrPrecisionOutOfRange` now says what it actually reports: a value with
  too many digits to fit in an int64 (returned by `MinorUnits` and
  `Money.AllocateEven`). Its old message, "precision must be >= 0", described
  a case that cannot happen, because `Context.Precision` is unsigned. The
  error variable is unchanged, so `errors.Is` checks keep working.

### Changed
- A `.gitattributes` file forces LF line endings, so `gofmt` and the tests
  give the same result on Windows checkouts as on Linux.

## [1.0.0] - 2026-09-14

### Changed
- Module path is now `github.com/movaltech/decimal`. No API changes from
  0.1.2.

## [0.1.3] - 2026-09-13

### Deprecated
- The previous module path was deprecated in favor of
  `github.com/movaltech/decimal`.

## [0.1.2] - 2026-09-08

### Fixed
- `Rescale` and `QuoScale` now reject a scale above `maxParseDigits`,
  matching the lower-bound check they already had. Previously, a single call
  with an oversized scale (e.g. `Rescale(math.MaxInt32, ...)`) would try to
  materialize a multi-gigabyte power of ten instead of returning
  `ErrScaleOutOfRange`.
- `New`, `NewFromBigInt`, and `NewFromMinorUnits` now panic if given a scale
  above `maxParseDigits`, for the same reason — left unchecked, the
  oversized scale wouldn't fail at construction, only later, as an
  out-of-memory allocation the first time the value was formatted or
  rescaled (`String()` in particular pads to the full scale with literal
  zero characters).
- `Pow` now also bounds the resulting coefficient's digit count, not just
  the resulting scale. The scale check alone (`d.Scale() * n`) never
  triggers for an integer base (`d.Scale() == 0`), so `NewFromInt64(2).Pow(n)`
  had no upper bound on `n` at all.

### Added
- `FuzzQuoScale`, `FuzzPow`, and `FuzzMoney` — fuzz targets alongside the
  existing `FuzzParse`/`FuzzRescale`, covering the rest of the surface that
  takes untrusted text or a caller-supplied scale/exponent. Each ran for
  millions of executions during development with no crashes found.
- A corpus of discovered fuzz cases for all five targets, committed under
  `testdata/fuzz/`, so `go test` replays them as ordinary subtests instead
  of only benefiting from a fresh, from-scratch fuzzing session.
- A GitHub Actions CI workflow (`.github/workflows/ci.yml`) that runs on
  every push and pull request: `gofmt`, `go vet`, a build, the full
  race-enabled test suite with a 100%-statement-coverage gate, a
  benchmark compile-check, and a 30-second smoke run of each fuzz target.

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

[0.1.2]: https://github.com/movaltech/decimal/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/movaltech/decimal/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/movaltech/decimal/releases/tag/v0.1.0
