package utility

import (
	"math/big"
	"strconv"
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
