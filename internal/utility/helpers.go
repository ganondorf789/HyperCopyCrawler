package utility

import (
	"math/big"
	"strconv"
	"strings"
	"time"
)

func WindowCutoff(window string) int64 {
	now := time.Now()
	switch window {
	case "day":
		return now.Add(-24 * time.Hour).UnixMilli()
	case "week":
		return now.Add(-7 * 24 * time.Hour).UnixMilli()
	case "month":
		return now.Add(-30 * 24 * time.Hour).UnixMilli()
	default:
		return 0
	}
}

func ParseBigFloat(s string) *big.Float {
	f, _, err := new(big.Float).Parse(s, 10)
	if err != nil {
		return nil
	}
	return f
}

func ParseBigFloatOr0(s string) *big.Float {
	f := ParseBigFloat(s)
	if f == nil {
		return new(big.Float)
	}
	return f
}

func AnyFloat(v interface{}) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case string:
		f, _ := strconv.ParseFloat(x, 64)
		return f
	default:
		return 0
	}
}

func FmtFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', 10, 64)
}

func OrZero(s string) string {
	if s == "" {
		return "0"
	}
	return s
}

func Abbr(addr string) string {
	if len(addr) > 10 {
		return addr[:10]
	}
	return addr
}

// AbbrWithEllipsis abbreviates s to 10 chars + "…" when longer.
func AbbrWithEllipsis(s string) string {
	if len(s) > 10 {
		return s[:10] + "…"
	}
	return s
}

// CalcResultSzi computes the resulting position after a fill:
// Buy (B): startPosition + sz, Sell (A): startPosition - sz
func CalcResultSzi(startPosition, sz, side string) string {
	start := ParseBigFloatOr0(startPosition)
	s := ParseBigFloatOr0(sz)
	var result *big.Float
	if side == "B" {
		result = new(big.Float).Add(start, s)
	} else {
		result = new(big.Float).Sub(start, s)
	}
	return result.Text('f', 8)
}

// MulStr multiplies two decimal strings and returns result with 8 decimal places.
func MulStr(a, b string) string {
	return new(big.Float).Mul(ParseBigFloatOr0(a), ParseBigFloatOr0(b)).Text('f', 8)
}

// SymbolAllowed checks if coin is allowed by symbolList (comma-separated) and symbolListType (WHITE/BLACK).
func SymbolAllowed(symbolList, symbolListType, coin string) bool {
	if symbolList == "" {
		return true
	}
	set := make(map[string]bool)
	for _, s := range strings.Split(symbolList, ",") {
		set[strings.TrimSpace(s)] = true
	}
	if symbolListType == "WHITE" {
		return set[coin]
	}
	return !set[coin]
}
