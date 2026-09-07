package decimal

import (
	"encoding/json"
	"math"
	"testing"
)

func TestClassicFloatFailure(t *testing.T) {
	// The canonical float64 footgun: 0.1 + 0.2 != 0.3 in binary floating
	// point. It must be exactly 0.3 here.
	a := MustParse("0.1")
	b := MustParse("0.2")
	got := a.Add(b)
	want := MustParse("0.3")
	if !got.Equal(want) {
		t.Fatalf("0.1 + 0.2 = %s, want %s", got, want)
	}
	if got.String() != "0.3" {
		t.Fatalf("String() = %q, want %q", got.String(), "0.3")
	}
}

func TestAddPreservesScale(t *testing.T) {
	a := MustParse("10.5")
	b := MustParse("0.25")
	got := a.Add(b)
	if got.String() != "10.75" {
		t.Fatalf("got %s, want 10.75", got)
	}
}

func TestSubNegative(t *testing.T) {
	got := MustParse("1.00").Sub(MustParse("1.50"))
	if got.String() != "-0.50" {
		t.Fatalf("got %s, want -0.50", got)
	}
}

func TestMulExact(t *testing.T) {
	// 19.99 * 3 must be exactly 59.97, not 59.96999999999999 (float64).
	price := MustParse("19.99")
	qty := NewFromInt64(3)
	got := price.Mul(qty)
	if got.String() != "59.97" {
		t.Fatalf("got %s, want 59.97", got)
	}
}

func TestQuoScaleHalfEven(t *testing.T) {
	cases := []struct {
		a, b string
		want string
	}{
		{"0.125", "1", "0.12"}, // tie -> even (2 is even)
		{"0.135", "1", "0.14"}, // tie -> even (4 is even)
		{"1", "4", "0.25"},     // exact
	}
	for _, c := range cases {
		a, b := MustParse(c.a), MustParse(c.b)
		got, err := a.QuoScale(b, 2, RoundHalfEven)
		if err != nil {
			t.Fatalf("QuoScale(%s,%s): %v", c.a, c.b, err)
		}
		if got.String() != c.want {
			t.Errorf("QuoScale(%s,%s) = %s, want %s", c.a, c.b, got, c.want)
		}
	}
}

func TestQuoScaleHalfUpVsHalfEven(t *testing.T) {
	a := MustParse("0.5")
	b := NewFromInt64(1)

	up, err := a.QuoScale(b, 0, RoundHalfUp)
	if err != nil {
		t.Fatal(err)
	}
	if up.String() != "1" {
		t.Errorf("HalfUp: got %s, want 1", up)
	}

	even, err := a.QuoScale(b, 0, RoundHalfEven)
	if err != nil {
		t.Fatal(err)
	}
	if even.String() != "0" {
		t.Errorf("HalfEven: got %s, want 0", even)
	}
}

func TestQuoScaleDivisionByZero(t *testing.T) {
	_, err := MustParse("1").QuoScale(Zero(), 2, RoundHalfUp)
	if err != ErrDivisionByZero {
		t.Fatalf("got %v, want ErrDivisionByZero", err)
	}
}

func TestQuoScaleRoundUnnecessaryFails(t *testing.T) {
	// 100/3 does not terminate at 2 decimal places.
	_, err := MustParse("100").QuoScale(NewFromInt64(3), 2, RoundUnnecessary)
	if err != ErrRoundingNecessary {
		t.Fatalf("got %v, want ErrRoundingNecessary", err)
	}
}

func TestQuoExact(t *testing.T) {
	got, err := MustParse("1").QuoExact(MustParse("4"))
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "0.25" {
		t.Fatalf("got %s, want 0.25", got)
	}

	_, err = MustParse("1").QuoExact(MustParse("3"))
	if err != ErrInexact {
		t.Fatalf("got %v, want ErrInexact", err)
	}
}

func TestQuoContext(t *testing.T) {
	ctx := Context{Precision: 10, Rounding: RoundHalfEven}
	got, err := MustParse("1").QuoContext(MustParse("3"), ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "0.3333333333" {
		t.Fatalf("got %s, want 0.3333333333", got)
	}
}

func TestRescaleRoundingModes(t *testing.T) {
	v := MustParse("2.345")
	cases := []struct {
		mode RoundingMode
		want string
	}{
		{RoundDown, "2.34"},
		{RoundUp, "2.35"},
		{RoundHalfUp, "2.35"},
		{RoundHalfEven, "2.34"}, // tie -> even (4)
		{RoundCeiling, "2.35"},
		{RoundFloor, "2.34"},
	}
	for _, c := range cases {
		got, err := v.Rescale(2, c.mode)
		if err != nil {
			t.Fatalf("%v: %v", c.mode, err)
		}
		if got.String() != c.want {
			t.Errorf("Rescale(%v) = %s, want %s", c.mode, got, c.want)
		}
	}
}

func TestRescaleNegativeCeilingFloor(t *testing.T) {
	v := MustParse("-2.345")
	ceil, _ := v.Rescale(2, RoundCeiling)
	if ceil.String() != "-2.34" {
		t.Errorf("Ceiling: got %s, want -2.34", ceil)
	}
	floor, _ := v.Rescale(2, RoundFloor)
	if floor.String() != "-2.35" {
		t.Errorf("Floor: got %s, want -2.35", floor)
	}
}

func TestCmpIgnoresTrailingScale(t *testing.T) {
	if MustParse("1.50").Cmp(MustParse("1.5")) != 0 {
		t.Fatal("1.50 should equal 1.5")
	}
	if MustParse("1.5").Cmp(MustParse("1.50")) != 0 {
		t.Fatal("1.5 should equal 1.50")
	}
}

func TestParseErrors(t *testing.T) {
	bad := []string{"", "-", ".", "1.2.3", "abc", "1e", "1.2e"}
	for _, s := range bad {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) succeeded, want error", s)
		}
	}
}

func TestParseExponent(t *testing.T) {
	got, err := Parse("1.5e2")
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "150" {
		t.Fatalf("got %s, want 150", got)
	}

	got2, err := Parse("150e-2")
	if err != nil {
		t.Fatal(err)
	}
	if got2.String() != "1.50" {
		t.Fatalf("got %s, want 1.50", got2)
	}
}

func TestJSONRoundTrip(t *testing.T) {
	type payload struct {
		Price Decimal `json:"price"`
	}
	p := payload{Price: MustParse("19.99")}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"price":"19.99"}` {
		t.Fatalf("got %s", b)
	}

	var out payload
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if !out.Price.Equal(p.Price) {
		t.Fatalf("round trip mismatch: got %s, want %s", out.Price, p.Price)
	}
}

func TestJSONUnmarshalBareNumber(t *testing.T) {
	var d Decimal
	if err := json.Unmarshal([]byte("10.50"), &d); err != nil {
		t.Fatal(err)
	}
	if d.String() != "10.50" {
		t.Fatalf("got %s, want 10.50", d)
	}
}

func TestMinorUnitsRoundTrip(t *testing.T) {
	d := NewFromMinorUnits(1050, 2)
	if d.String() != "10.50" {
		t.Fatalf("got %s, want 10.50", d)
	}
	units, err := d.MinorUnits(2, RoundUnnecessary)
	if err != nil {
		t.Fatal(err)
	}
	if units != 1050 {
		t.Fatalf("got %d, want 1050", units)
	}
}

func TestMoneyCurrencyMismatch(t *testing.T) {
	usd, _ := ParseMoney("10.00", "USD", RoundHalfEven)
	eur, _ := ParseMoney("10.00", "EUR", RoundHalfEven)
	if _, err := usd.Add(eur); err != ErrCurrencyMismatch {
		t.Fatalf("got %v, want ErrCurrencyMismatch", err)
	}
}

func TestMoneyAllocateEvenReconstructsTotal(t *testing.T) {
	// $100.00 split 3 ways must sum back to exactly $100.00, distributing
	// the leftover cent instead of losing it to independent rounding.
	total, _ := ParseMoney("100.00", "USD", RoundHalfEven)
	shares, err := total.AllocateEven(3)
	if err != nil {
		t.Fatal(err)
	}
	if len(shares) != 3 {
		t.Fatalf("got %d shares, want 3", len(shares))
	}
	sum := shares[0].Amount.Add(shares[1].Amount).Add(shares[2].Amount)
	if !sum.Equal(total.Amount) {
		t.Fatalf("shares sum to %s, want %s", sum, total.Amount)
	}
	// Expect 33.34 / 33.33 / 33.33 (leftover cent to the first share).
	if shares[0].Amount.String() != "33.34" {
		t.Errorf("share[0] = %s, want 33.34", shares[0].Amount)
	}
	if shares[1].Amount.String() != "33.33" {
		t.Errorf("share[1] = %s, want 33.33", shares[1].Amount)
	}
}

func TestMoneyJPYZeroMinorDigits(t *testing.T) {
	m, err := ParseMoney("500", "JPY", RoundHalfEven)
	if err != nil {
		t.Fatal(err)
	}
	if m.Amount.String() != "500" {
		t.Fatalf("got %s, want 500", m.Amount)
	}
}

func TestContextRoundSignificantDigits(t *testing.T) {
	ctx := Context{Precision: 4, Rounding: RoundHalfUp}
	got, err := ctx.Round(MustParse("123.456"))
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "123.5" {
		t.Fatalf("got %s, want 123.5", got)
	}
}

func TestContextRoundIntegerPartExceedsPrecision(t *testing.T) {
	// Regression test: Round used to silently return the value unchanged
	// whenever the *integer* part alone had more digits than Precision,
	// because Rescale cannot express a negative scale. Round must instead
	// carry the rounding into the integer part and re-pad with zeros.
	ctx := Context{Precision: 3, Rounding: RoundHalfUp}

	got, err := ctx.Round(MustParse("123456"))
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "123000" {
		t.Fatalf("Round(123456, prec=3) = %s, want 123000", got)
	}

	// All-nines carry: 999999 -> 1000 (3 sig digits) -> re-padded to 1000000.
	got2, err := ctx.Round(MustParse("999999"))
	if err != nil {
		t.Fatal(err)
	}
	if got2.String() != "1000000" {
		t.Fatalf("Round(999999, prec=3) = %s, want 1000000", got2)
	}
}

func TestParseHugeExponentRejected(t *testing.T) {
	// Regression test: a short scientific-notation literal used to make
	// Parse expand the coefficient into an unbounded number of trailing
	// zeros (e.g. "1e50000000" allocates and scans tens of megabytes from a
	// dozen input bytes), a DoS vector since Parse is the documented entry
	// point for untrusted external text. It must now fail fast instead.
	_, err := Parse("1e999999999")
	if err != ErrExponentOutOfRange {
		t.Fatalf("got %v, want ErrExponentOutOfRange", err)
	}

	// A large but bounded exponent must still work.
	got, err := Parse("1e5")
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "100000" {
		t.Fatalf("got %s, want 100000", got)
	}
}

func TestParseHugeNegativeExponentRejected(t *testing.T) {
	// Regression test: a huge *negative* exponent (e.g. "1e-999999999")
	// doesn't allocate inside Parse itself, but produces a Decimal with a
	// billion-digit scale that later hangs/OOMs the first time String() (or
	// MarshalJSON, Rescale, ...) tries to render it. This must be rejected
	// up front, symmetrically with the positive-exponent case.
	_, err := Parse("1e-999999999")
	if err != ErrExponentOutOfRange {
		t.Fatalf("got %v, want ErrExponentOutOfRange", err)
	}
}

func TestComparisonSugar(t *testing.T) {
	a := MustParse("1.5")
	b := MustParse("2.5")
	if !a.LessThan(b) || b.LessThan(a) {
		t.Fatal("LessThan failed")
	}
	if !b.GreaterThan(a) || a.GreaterThan(b) {
		t.Fatal("GreaterThan failed")
	}
	if !a.LessThanOrEqual(a) || !a.LessThanOrEqual(b) || b.LessThanOrEqual(a) {
		t.Fatal("LessThanOrEqual failed")
	}
	if !a.GreaterThanOrEqual(a) || !b.GreaterThanOrEqual(a) || a.GreaterThanOrEqual(b) {
		t.Fatal("GreaterThanOrEqual failed")
	}
	if !MustParse("-1").IsNegative() || MustParse("1").IsNegative() || Zero().IsNegative() {
		t.Fatal("IsNegative failed")
	}
	if !MustParse("1").IsPositive() || MustParse("-1").IsPositive() || Zero().IsPositive() {
		t.Fatal("IsPositive failed")
	}
}

func TestTextMarshalRoundTrip(t *testing.T) {
	d := MustParse("19.99")
	text, err := d.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	if string(text) != "19.99" {
		t.Fatalf("MarshalText() = %q, want %q", text, "19.99")
	}

	var out Decimal
	if err := out.UnmarshalText(text); err != nil {
		t.Fatal(err)
	}
	if !out.Equal(d) {
		t.Fatalf("round trip mismatch: got %s, want %s", out, d)
	}

	var zero Decimal
	if err := zero.UnmarshalText(nil); err != nil || !zero.IsZero() {
		t.Fatalf("UnmarshalText(nil) should reset to zero, got %s, err=%v", zero, err)
	}
}

func TestTextMarshalAsMapKey(t *testing.T) {
	// encoding/json uses TextMarshaler/TextUnmarshaler for map keys, so this
	// is the concrete reason MarshalText/UnmarshalText exist beyond
	// MarshalJSON/UnmarshalJSON. The round-tripped keys are looked up by
	// String(), not by re-indexing with a freshly parsed Decimal: per the
	// Decimal type doc, two independently-parsed equal values are not == to
	// each other, so a fresh MustParse("1.50") is not a valid map key for an
	// entry that was itself decoded via UnmarshalText.
	m := map[Decimal]string{MustParse("1.50"): "a", MustParse("2.00"): "b"}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var out map[Decimal]string
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for k, v := range out {
		got[k.String()] = v
	}
	want := map[string]string{"1.50": "a", "2.00": "b"}
	if len(got) != len(want) || got["1.50"] != "a" || got["2.00"] != "b" {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestSignAndAbs(t *testing.T) {
	if MustParse("-5.00").Sign() != -1 {
		t.Fatal("sign should be -1")
	}
	if MustParse("-5.00").Abs().String() != "5.00" {
		t.Fatal("abs should be 5.00")
	}
	if !Zero().IsZero() {
		t.Fatal("Zero() should be zero")
	}
}

// ---------------------------------------------------------------------
// The tests below close the remaining coverage gaps: each one targets a
// specific error path or edge branch that the tests above never reached.
// ---------------------------------------------------------------------

func TestRescaleNegativeScaleRejected(t *testing.T) {
	_, err := MustParse("1.23").Rescale(-1, RoundHalfUp)
	if err != ErrScaleOutOfRange {
		t.Fatalf("got %v, want ErrScaleOutOfRange", err)
	}
}

func TestRescaleRoundUnnecessaryFails(t *testing.T) {
	// 2.345 cannot be rescaled to 2 places without discarding a non-zero
	// digit, so RoundUnnecessary must reject it instead of rounding silently.
	_, err := MustParse("2.345").Rescale(2, RoundUnnecessary)
	if err != ErrRoundingNecessary {
		t.Fatalf("got %v, want ErrRoundingNecessary", err)
	}
}

func TestQuoScaleNegativeScaleRejected(t *testing.T) {
	_, err := MustParse("1").QuoScale(MustParse("3"), -1, RoundHalfUp)
	if err != ErrScaleOutOfRange {
		t.Fatalf("got %v, want ErrScaleOutOfRange", err)
	}
}

func TestQuoExactDivisionByZero(t *testing.T) {
	_, err := MustParse("1").QuoExact(Zero())
	if err != ErrDivisionByZero {
		t.Fatalf("got %v, want ErrDivisionByZero", err)
	}
}

func TestQuoExactZeroDividend(t *testing.T) {
	got, err := Zero().QuoExact(MustParse("5"))
	if err != nil {
		t.Fatal(err)
	}
	if !got.IsZero() {
		t.Fatalf("got %s, want 0", got)
	}
}

func TestContextRoundUnlimitedPrecision(t *testing.T) {
	// Precision 0 means unlimited: Round must return d unchanged.
	ctx := Context{Precision: 0, Rounding: RoundHalfUp}
	got, err := ctx.Round(MustParse("123.456789"))
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "123.456789" {
		t.Fatalf("got %s, want 123.456789", got)
	}
}

func TestContextRoundAlreadyWithinPrecision(t *testing.T) {
	// Zero has 1 digit by definition, well within any positive precision, so
	// Round must return it unchanged without touching the rounding path.
	ctx := Context{Precision: 3, Rounding: RoundHalfUp}
	got, err := ctx.Round(Zero())
	if err != nil {
		t.Fatal(err)
	}
	if !got.IsZero() {
		t.Fatalf("got %s, want 0", got)
	}
}

func TestContextRoundUnnecessaryFails(t *testing.T) {
	ctx := Context{Precision: 2, Rounding: RoundUnnecessary}
	_, err := ctx.Round(MustParse("123"))
	if err != ErrRoundingNecessary {
		t.Fatalf("got %v, want ErrRoundingNecessary", err)
	}
}

func TestContextRoundNegativeIntegerOverflowsPrecision(t *testing.T) {
	// Mirrors TestContextRoundIntegerPartExceedsPrecision but for a negative
	// value, to exercise both the sign-restoring branch in Round and the
	// negative-coefficient branch of numDigits.
	ctx := Context{Precision: 3, Rounding: RoundHalfUp}
	got, err := ctx.Round(MustParse("-123456"))
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "-123000" {
		t.Fatalf("got %s, want -123000", got)
	}
}

func TestQuoContextDivisionByZero(t *testing.T) {
	_, err := MustParse("1").QuoContext(Zero(), DefaultContext)
	if err != ErrDivisionByZero {
		t.Fatalf("got %v, want ErrDivisionByZero", err)
	}
}

func TestQuoContextZeroPrecisionUsesDefault(t *testing.T) {
	ctx := Context{Precision: 0, Rounding: RoundHalfEven}
	got, err := MustParse("1").QuoContext(MustParse("3"), ctx)
	if err != nil {
		t.Fatal(err)
	}
	if want := int32(DefaultContext.Precision); got.Scale() != want {
		t.Fatalf("got scale %d, want %d (DefaultContext precision applied)", got.Scale(), want)
	}
}

func TestQuoContextPropagatesUnderlyingError(t *testing.T) {
	// Context.Precision is a uint32 so it accepts values that overflow int32
	// once QuoContext converts it; a value at the top half of the uint32
	// range (here, 3e9) wraps around to a large *negative* int32, which
	// drives the internal working scale negative — QuoScale correctly
	// rejects that. This test locks in that QuoContext surfaces the error
	// instead of silently miscomputing.
	ctx := Context{Precision: 3_000_000_000, Rounding: RoundHalfEven}
	_, err := MustParse("1").QuoContext(MustParse("3"), ctx)
	if err != ErrScaleOutOfRange {
		t.Fatalf("got %v, want ErrScaleOutOfRange", err)
	}
}

func TestMinorUnitsRescaleError(t *testing.T) {
	_, err := MustParse("1.005").MinorUnits(2, RoundUnnecessary)
	if err != ErrRoundingNecessary {
		t.Fatalf("got %v, want ErrRoundingNecessary", err)
	}
}

func TestMinorUnitsOverflowsInt64(t *testing.T) {
	huge := MustParse("123456789012345678.90")
	_, err := huge.MinorUnits(2, RoundDown)
	if err != ErrPrecisionOutOfRange {
		t.Fatalf("got %v, want ErrPrecisionOutOfRange", err)
	}
}

func TestNewWithNegativeScaleNormalizedToZero(t *testing.T) {
	got := New(5, -3)
	if got.Scale() != 0 {
		t.Fatalf("Scale() = %d, want 0", got.Scale())
	}
	if got.String() != "5" {
		t.Fatalf("got %s, want 5", got)
	}
}

func TestParseInvalidExponentDigits(t *testing.T) {
	_, err := Parse("1eX")
	if err != ErrInvalidSyntax {
		t.Fatalf("got %v, want ErrInvalidSyntax", err)
	}
}

func TestMustParsePanicsOnInvalidInput(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("MustParse did not panic on invalid input")
		}
	}()
	MustParse("not-a-number")
}

func TestUnmarshalJSONEmptyInput(t *testing.T) {
	var d Decimal
	if err := d.UnmarshalJSON(nil); err != ErrInvalidSyntax {
		t.Fatalf("got %v, want ErrInvalidSyntax", err)
	}
}

func TestUnmarshalJSONNull(t *testing.T) {
	d := MustParse("1.23")
	if err := json.Unmarshal([]byte("null"), &d); err != nil {
		t.Fatal(err)
	}
	if !d.IsZero() {
		t.Fatalf("got %s, want 0 (null resets to zero)", d)
	}
}

func TestUnmarshalJSONInvalidContent(t *testing.T) {
	var d Decimal
	if err := json.Unmarshal([]byte(`"not-a-number"`), &d); err == nil {
		t.Fatal("expected an error for invalid decimal text")
	}
}

func TestUnmarshalTextInvalidContent(t *testing.T) {
	var d Decimal
	if err := d.UnmarshalText([]byte("not-a-number")); err == nil {
		t.Fatal("expected an error for invalid decimal text")
	}
}

func TestScanStringError(t *testing.T) {
	var d Decimal
	if err := d.Scan("not-a-number"); err == nil {
		t.Fatal("expected an error for invalid decimal text")
	}
}

func TestScanBytesError(t *testing.T) {
	var d Decimal
	if err := d.Scan([]byte("not-a-number")); err == nil {
		t.Fatal("expected an error for invalid decimal text")
	}
}

func TestScanFloatError(t *testing.T) {
	// fmt.Sprintf("%v", NaN) is "NaN", which Parse correctly rejects: Scan
	// cannot invent digits a float64 source never had.
	var d Decimal
	if err := d.Scan(math.NaN()); err == nil {
		t.Fatal("expected an error for NaN")
	}
}

func TestScanUnsupportedType(t *testing.T) {
	var d Decimal
	err := d.Scan(true)
	if err == nil {
		t.Fatal("expected an error for an unsupported Scan source type")
	}
}

func TestNewMoneyRescaleError(t *testing.T) {
	_, err := NewMoney(MustParse("1.005"), "USD", RoundUnnecessary)
	if err != ErrRoundingNecessary {
		t.Fatalf("got %v, want ErrRoundingNecessary", err)
	}
}

func TestParseMoneyParseError(t *testing.T) {
	_, err := ParseMoney("not-a-number", "USD", RoundHalfEven)
	if err != ErrInvalidSyntax {
		t.Fatalf("got %v, want ErrInvalidSyntax", err)
	}
}

func TestMoneyAddHappyPath(t *testing.T) {
	a, _ := ParseMoney("2.50", "USD", RoundHalfEven)
	b, _ := ParseMoney("1.25", "USD", RoundHalfEven)
	sum, err := a.Add(b)
	if err != nil {
		t.Fatal(err)
	}
	if sum.String() != "3.75 USD" {
		t.Fatalf("got %s, want 3.75 USD", sum)
	}
}

func TestMoneySubCurrencyMismatch(t *testing.T) {
	usd, _ := ParseMoney("10.00", "USD", RoundHalfEven)
	eur, _ := ParseMoney("1.00", "EUR", RoundHalfEven)
	if _, err := usd.Sub(eur); err != ErrCurrencyMismatch {
		t.Fatalf("got %v, want ErrCurrencyMismatch", err)
	}
}

func TestAllocateEvenRejectsNonPositiveN(t *testing.T) {
	m, _ := ParseMoney("10.00", "USD", RoundHalfEven)
	if _, err := m.AllocateEven(0); err == nil {
		t.Fatal("expected an error for n=0")
	}
	if _, err := m.AllocateEven(-1); err == nil {
		t.Fatal("expected an error for n=-1")
	}
}

func TestAllocateEvenMinorUnitsOverflow(t *testing.T) {
	huge, _ := ParseMoney("123456789012345678.90", "USD", RoundHalfEven)
	if _, err := huge.AllocateEven(3); err != ErrPrecisionOutOfRange {
		t.Fatalf("got %v, want ErrPrecisionOutOfRange", err)
	}
}

func TestRoundingModeStringAllValues(t *testing.T) {
	cases := map[RoundingMode]string{
		RoundHalfEven:    "HalfEven",
		RoundHalfUp:      "HalfUp",
		RoundHalfDown:    "HalfDown",
		RoundUp:          "Up",
		RoundDown:        "Down",
		RoundCeiling:     "Ceiling",
		RoundFloor:       "Floor",
		RoundUnnecessary: "Unnecessary",
	}
	for mode, want := range cases {
		if got := mode.String(); got != want {
			t.Errorf("%d.String() = %q, want %q", mode, got, want)
		}
	}
	if got := RoundingMode(99).String(); got != "RoundingMode(?)" {
		t.Fatalf("got %q, want %q", got, "RoundingMode(?)")
	}
}

func TestApplyRoundingHalfDownTie(t *testing.T) {
	// 0.5 is an exact tie; RoundHalfDown breaks it toward zero.
	got, err := MustParse("0.5").QuoScale(NewFromInt64(1), 0, RoundHalfDown)
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "0" {
		t.Fatalf("got %s, want 0", got)
	}
}

func TestApplyRoundingUnknownMode(t *testing.T) {
	// An out-of-range RoundingMode value falls through to the same
	// conservative behavior as RoundUnnecessary: refuse to guess.
	_, err := MustParse("1").QuoScale(MustParse("3"), 2, RoundingMode(99))
	if err != ErrRoundingNecessary {
		t.Fatalf("got %v, want ErrRoundingNecessary", err)
	}
}
