package decimal_test

import (
	"testing"

	"github.com/diegofxm/decimal"
)

// These benchmarks exist because Decimal is backed by *big.Int, which
// allocates on essentially every operation — a cost that is easy to overlook
// given the type's 100%-covered, exact-arithmetic guarantees say nothing
// about throughput. Run with: go test -bench=. -benchmem ./...

func BenchmarkAdd(b *testing.B) {
	x := decimal.MustParse("123456.789")
	y := decimal.MustParse("987.654321")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = x.Add(y)
	}
}

func BenchmarkMul(b *testing.B) {
	x := decimal.MustParse("123456.789")
	y := decimal.MustParse("987.654321")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = x.Mul(y)
	}
}

func BenchmarkQuoScale(b *testing.B) {
	x := decimal.MustParse("100.00")
	y := decimal.NewFromInt64(3)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = x.QuoScale(y, 2, decimal.RoundHalfEven)
	}
}

func BenchmarkParse(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = decimal.Parse("-123456.789012")
	}
}

func BenchmarkString(b *testing.B) {
	x := decimal.MustParse("-123456.789012")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = x.String()
	}
}

func BenchmarkPow(b *testing.B) {
	rate := decimal.MustParse("1.05")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = rate.Pow(12)
	}
}
