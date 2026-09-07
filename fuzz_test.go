package decimal

import "testing"

// FuzzParse checks that Parse never panics and never hangs on arbitrary
// input, and that whenever it succeeds, the result round-trips through
// String()+Parse to the same value. Parse is the package's documented entry
// point for untrusted external text (user input, JSON, CSV), so it is the
// one function in this package that must survive adversarial fuzzing, not
// just hand-picked test cases.
//
// Run with: go test -fuzz=FuzzParse -fuzztime=60s
func FuzzParse(f *testing.F) {
	seeds := []string{
		"0", "1", "-1", "0.1", "-0.1", "1.5e2", "150e-2", "1e999999999",
		"", "-", ".", "1.2.3", "abc", "1e", "1.2e", "+5", "5e0", "0.000",
		"999999999999999999999999999999", "1e-999999999", "1e-2000000",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		d, err := Parse(s)
		if err != nil {
			return
		}
		back, err := Parse(d.String())
		if err != nil {
			t.Fatalf("Parse(%q) succeeded as %s, but Parse(String()) failed: %v", s, d, err)
		}
		if !back.Equal(d) {
			t.Fatalf("round-trip mismatch: Parse(%q) = %s, Parse(that.String()=%q) = %s", s, d, d.String(), back)
		}
	})
}

// FuzzRescale checks that Rescale never panics for any coefficient/scale
// combination reachable via Parse, across every rounding mode.
func FuzzRescale(f *testing.F) {
	f.Add("123.456", int32(2), uint8(RoundHalfEven))
	f.Add("0", int32(0), uint8(RoundUp))
	f.Add("-999.999", int32(0), uint8(RoundFloor))

	f.Fuzz(func(t *testing.T, s string, newScale int32, mode uint8) {
		d, err := Parse(s)
		if err != nil {
			return
		}
		if mode > uint8(RoundUnnecessary) {
			return
		}
		_, _ = d.Rescale(newScale, RoundingMode(mode))
	})
}
