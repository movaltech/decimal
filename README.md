# decimal

[![Go Reference](https://pkg.go.dev/badge/github.com/diegofxm/decimal.svg)](https://pkg.go.dev/github.com/diegofxm/decimal)
[![Go Version](https://img.shields.io/github/go-mod/go-version/diegofxm/decimal)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Exact, arbitrary-precision fixed-point decimal arithmetic for Go. Built for
money and any other value where binary floating-point rounding is not an
option.

```go
a := decimal.MustParse("0.1")
b := decimal.MustParse("0.2")
a.Add(b).String() // "0.3" — exact, not "0.30000000000000004"
```

## Overview

Go has no built-in decimal type. Anything that touches money — an order
total, an invoice line, a payout — either goes through `float64`, which
cannot represent most decimal fractions exactly and silently accumulates
rounding error, or through a hand-rolled integer-cents scheme that has to be
reinvented on every project.

`decimal` fills that gap with a single, dependency-free type: a coefficient
(`*big.Int`) and a scale, so every value is represented exactly, at whatever
precision it needs, and every arithmetic operation states up front how
rounding — if any — is resolved. Nothing in this package ever routes a value
through `float64`.

## Features

- **Exact arithmetic.** `Add`, `Sub`, and `Mul` never lose precision.
- **Explicit rounding, always.** Division requires you to say how many
  decimal places you want and how to resolve the remainder — there is no
  silent, implicit rounding anywhere in the API.
- **Arbitrary precision.** Backed by `math/big`, so precision is bounded by
  memory, not by a fixed number of bits.
- **A rounding mode that fails on purpose.** `RoundUnnecessary` returns an
  error instead of a number when an operation would actually need to round,
  so precision bugs surface immediately instead of silently drifting.
- **Currency-aware `Money`.** Pairs an amount with an ISO 4217 currency code
  and refuses to combine mismatched currencies; `AllocateEven` splits a total
  into shares that always sum back to the original, distributing the
  leftover minor unit instead of dropping it.
- **Immutable.** Every method returns a new value. A `Decimal` is safe to
  copy, compare, and share across goroutines.
- **Safe serialization.** Implements `json.Marshaler`/`Unmarshaler`,
  `encoding.TextMarshaler`/`TextUnmarshaler`, and
  `database/sql/driver.Valuer`/`Scanner` — none of them ever pass through
  `float64`.
- **Hardened parsing.** `Parse` is the intended entry point for untrusted
  external text (user input, JSON, CSV) and bounds pathological input (e.g.
  a scientific-notation exponent designed to blow up memory) instead of
  trusting it.
- **Verified, not just tested.** 100% statement coverage, clean under the
  race detector, a property-based test that checks `QuoScale` against
  `math/big.Rat` as an independent source of truth across thousands of
  random cases, and fuzz tests (`FuzzParse`, `FuzzRescale`) targeting the
  parser specifically because it is the untrusted-input boundary.

## Installation

```
go get github.com/diegofxm/decimal
```

## Quick start

`float64` doesn't fail on every input — which is exactly what makes it
dangerous to use for money: it looks fine until, on some input you didn't
hand-pick, it isn't. All three of these are ordinary line-item math, computed
with real `float64` variables (not literal constants, which Go's compiler
can fold at arbitrary precision and print misleadingly cleanly):

```go
var price, qty float64 = 2.32, 3
fmt.Println(price * qty) // 6.959999999999999 — not 6.96

d := decimal.MustParse("2.32").Mul(decimal.NewFromInt64(3))
fmt.Println(d) // 6.96 — exact
```

```go
var a, b float64 = 1.03, 0.42
fmt.Println(a - b) // 0.6100000000000001 — not 0.61

d := decimal.MustParse("1.03").Sub(decimal.MustParse("0.42"))
fmt.Println(d) // 0.61 — exact
```

```go
var total float64
for i := 0; i < 20; i++ {
    total += 29.99 // 20 invoice lines at the same price
}
fmt.Println(total) // 599.8000000000001 — not 599.80

sum := decimal.Zero()
for i := 0; i < 20; i++ {
    sum = sum.Add(decimal.MustParse("29.99"))
}
fmt.Println(sum) // 599.80 — exact, no matter how many lines
```

None of these are edge cases — they're ordinary arithmetic a billing system
runs constantly. The error doesn't announce itself with a crash; it just
sits in the trailing digits until a total doesn't reconcile and someone has
to go find out why.

Division works the same way, but the rounding decision is explicit instead
of implicit:

```go
price := decimal.MustParse("19.99")
qty := decimal.NewFromInt64(3)
total := price.Mul(qty) // exact: 59.97

// Division always requires an explicit number of decimal places and a
// rounding mode — there is no implicit, "silent" division.
perUnit, err := total.QuoScale(qty, 2, decimal.RoundHalfEven)
```

## Three ways to divide, on purpose

A single `Div` method would have to guess what you want, and guessing is
exactly what causes precision bugs. `decimal` gives you three explicit
options instead:

- **`QuoScale(divisor, scale, mode)`** — for money. You decide the exact
  number of decimal places the result must have; the remainder is resolved
  using `mode`.
- **`QuoContext(divisor, ctx)`** — for general numeric computation. Rounds to
  a fixed number of *significant* digits rather than a fixed number of
  decimal places, via a `Context{Precision, Rounding}`.
- **`QuoExact(divisor)`** — fails with `ErrInexact` if the division does not
  terminate within `MaxExactScale` digits, instead of rounding silently.
  Useful for asserting that a split is exact.

```go
one := decimal.NewFromInt64(1)
three := decimal.NewFromInt64(3)

one.QuoScale(three, 2, decimal.RoundHalfUp)   // "0.33"
one.QuoContext(three, decimal.DefaultContext) // 28 significant digits
one.QuoExact(three)                           // error: ErrInexact (1/3 never terminates)
```

## Rounding modes

`RoundHalfEven` (banker's rounding — the recommended default, since it does
not bias a long-running sum up or down), `RoundHalfUp`, `RoundHalfDown`,
`RoundUp`, `RoundDown`, `RoundCeiling`, `RoundFloor`, and
`RoundUnnecessary`, which fails instead of rounding at all — use it wherever
an inexact result should be treated as a bug, not a number.

## Money with a currency (`Money`)

```go
usd, _ := decimal.ParseMoney("100.00", "USD", decimal.RoundHalfEven)
eur, _ := decimal.ParseMoney("10.00", "EUR", decimal.RoundHalfEven)

_, err := usd.Add(eur) // ErrCurrencyMismatch — it will not convert currencies for you

shares, _ := usd.AllocateEven(3) // 33.34 / 33.33 / 33.33 — sums back to exactly 100.00
```

`AllocateEven` distributes the leftover minor unit instead of rounding each
share independently, which is the classic bug that leaves split totals not
summing back to the original amount.

## Serialization without `float64`

```go
type Order struct {
    Total decimal.Decimal `json:"total"`
}
json.Marshal(Order{Total: decimal.MustParse("19.99")})
// {"total":"19.99"} — a quoted string, not a bare number, so nothing on the
// receiving end can decode it into a float64 by accident.
```

`Decimal` also implements `database/sql/driver.Valuer` and `sql.Scanner` —
pair it with a `NUMERIC`/`DECIMAL` column, never `FLOAT`/`DOUBLE`, or the
precision gap simply moves into the database — and
`encoding.TextMarshaler`/`TextUnmarshaler`, which covers everything that
isn't `encoding/json` (YAML, TOML, map keys).

> **Note:** `Decimal` holds an internal `*big.Int`, so it must not be
> compared with `==` or used directly as a map key — two independently
> parsed equal values are not guaranteed to be `==`. Use `Equal`/`Cmp` for
> comparisons, and `d.String()` if you need a comparable key.

## What it deliberately does not do

- **No operator overloading.** Go doesn't allow it; use `.Add()`, `.Sub()`,
  `.Mul()`, and friends.
- **No currency conversion.** `Money` validates that currencies match; it
  will never guess an exchange rate for you.
- **No implicit float64 interop.** `Float64()` exists solely to interoperate
  with non-monetary APIs (plotting, metrics) and is documented as lossy —
  never use it on a money code path.

## Testing

```
go test -race -cover ./...
```

This includes a property-based test (`property_test.go`) that checks
`QuoScale` against `math/big.Rat` — an independent mathematical ground truth
— across thousands of random cases, on top of the classic hand-picked ones
(`0.1 + 0.2`, `19.99 * 3`, splitting a total three ways, banker's-rounding
ties, and so on). `fuzz_test.go` fuzzes `Parse` and `Rescale` directly, since
`Parse` is the package's boundary with untrusted external input.

## Documentation

Full API reference: [pkg.go.dev/github.com/diegofxm/decimal](https://pkg.go.dev/github.com/diegofxm/decimal)

## Why this exists

This package was born out of a gap, not a preference: Go's standard library
has no fixed-point decimal type, and the third-party options in the
ecosystem don't agree on an API or on how rounding should behave by default.
Anyone building a payments system, a ledger, or anything that bills a real
person ends up solving the same problem — exact representation, explicit
rounding, safe serialization — from scratch, or trusting `float64` and
hoping the rounding error stays small enough not to matter. It doesn't stay
small enough. This is the version of that solution meant to be shared,
rather than rewritten in every codebase that needs it.

## Contributing

Issues and pull requests are welcome. If you're changing rounding or parsing
behavior, please add a test case to `decimal_test.go` or `property_test.go`
— those two files are what this package's correctness claims actually rest
on.

## License

MIT — see [LICENSE](LICENSE).
