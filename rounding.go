package decimal

import "math/big"

// RoundingMode selects how a Decimal is rounded when the exact mathematical
// result cannot be represented at the target scale or precision.
//
// Every rounding decision in this package goes through one of these named
// modes — there is no hidden default that rounds one way in one code path
// and a different way somewhere else.
type RoundingMode uint8

const (
	// RoundHalfEven rounds to the nearest neighbor, and on an exact tie
	// rounds to the neighbor with an even last digit ("banker's rounding").
	// This is the mode required by IEEE 754-2008 decimal arithmetic, and the
	// recommended default for repeated aggregation because it does not bias
	// the sum up or down over many roundings.
	RoundHalfEven RoundingMode = iota

	// RoundHalfUp rounds to the nearest neighbor, and on an exact tie rounds
	// away from zero (1.5 -> 2, -1.5 -> -2). This is the "textbook" rounding
	// most people mean when they say "round half up".
	RoundHalfUp

	// RoundHalfDown rounds to the nearest neighbor, and on an exact tie
	// rounds toward zero (1.5 -> 1, -1.5 -> -1).
	RoundHalfDown

	// RoundUp rounds away from zero regardless of the discarded digits.
	RoundUp

	// RoundDown truncates toward zero, discarding any fractional remainder.
	RoundDown

	// RoundCeiling rounds toward positive infinity.
	RoundCeiling

	// RoundFloor rounds toward negative infinity.
	RoundFloor

	// RoundUnnecessary asserts that the operation is exact. If rounding
	// would actually be required, the operation fails with
	// ErrRoundingNecessary instead of silently discarding precision. Use
	// this for money paths where an inexact result indicates a bug (e.g.
	// splitting a total into shares that must sum back exactly).
	RoundUnnecessary
)

func (m RoundingMode) String() string {
	switch m {
	case RoundHalfEven:
		return "HalfEven"
	case RoundHalfUp:
		return "HalfUp"
	case RoundHalfDown:
		return "HalfDown"
	case RoundUp:
		return "Up"
	case RoundDown:
		return "Down"
	case RoundCeiling:
		return "Ceiling"
	case RoundFloor:
		return "Floor"
	case RoundUnnecessary:
		return "Unnecessary"
	default:
		return "RoundingMode(?)"
	}
}

// applyRounding decides whether the truncated quotient q must be incremented
// by one, given the non-negative remainder r of dividing by the positive
// divisor div, the sign of the true (untruncated) result, and the rounding
// mode. q, r and div must all be non-negative; sign of the result is passed
// separately in neg.
//
// It returns the (possibly incremented) quotient, or ErrRoundingNecessary if
// mode is RoundUnnecessary and r != 0.
func applyRounding(q, r, div *big.Int, neg bool, mode RoundingMode) (*big.Int, error) {
	if r.Sign() == 0 {
		// Exact result: every mode agrees, nothing to decide.
		return q, nil
	}

	switch mode {
	case RoundUnnecessary:
		return nil, ErrRoundingNecessary

	case RoundDown:
		return q, nil

	case RoundUp:
		return new(big.Int).Add(q, big.NewInt(1)), nil

	case RoundCeiling:
		if !neg {
			return new(big.Int).Add(q, big.NewInt(1)), nil
		}
		return q, nil

	case RoundFloor:
		if neg {
			return new(big.Int).Add(q, big.NewInt(1)), nil
		}
		return q, nil

	case RoundHalfUp, RoundHalfDown, RoundHalfEven:
		twiceR := new(big.Int).Lsh(r, 1) // 2*r, compared against div to locate the halfway point
		cmp := twiceR.Cmp(div)
		switch {
		case cmp < 0: // remainder < half: always round down
			return q, nil
		case cmp > 0: // remainder > half: always round up
			return new(big.Int).Add(q, big.NewInt(1)), nil
		default: // remainder == half: mode-specific tie break
			switch mode {
			case RoundHalfUp:
				return new(big.Int).Add(q, big.NewInt(1)), nil
			case RoundHalfDown:
				return q, nil
			default: // RoundHalfEven
				odd := new(big.Int).And(q, big.NewInt(1)).Sign() != 0
				if odd {
					return new(big.Int).Add(q, big.NewInt(1)), nil
				}
				return q, nil
			}
		}

	default:
		return nil, ErrRoundingNecessary
	}
}
