package decimal_test

import (
	"fmt"

	"github.com/movaltech/decimal"
)

func ExampleParse() {
	d, err := decimal.Parse("19.99")
	fmt.Println(d, err)
	// Output: 19.99 <nil>
}

func ExampleDecimal_Add() {
	a := decimal.MustParse("0.1")
	b := decimal.MustParse("0.2")
	fmt.Println(a.Add(b))
	// Output: 0.3
}

func ExampleDecimal_Pow() {
	rate := decimal.MustParse("1.05") // 5% interest per period
	compounded, err := rate.Pow(2)
	fmt.Println(compounded, err)
	// Output: 1.1025 <nil>
}

func ExampleMoney_AllocateEven() {
	total, _ := decimal.ParseMoney("100.00", "USD", decimal.RoundHalfEven)
	shares, _ := total.AllocateEven(3)
	fmt.Println(shares)
	// Output: [33.34 USD 33.33 USD 33.33 USD]
}

func ExampleMoney_Mul() {
	unitPrice, _ := decimal.ParseMoney("19.99", "USD", decimal.RoundHalfEven)
	lineTotal := unitPrice.Mul(decimal.NewFromInt64(3))
	fmt.Println(lineTotal)
	// Output: 59.97 USD
}
