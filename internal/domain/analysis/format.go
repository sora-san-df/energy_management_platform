package analysis

import (
	"math"
	"strconv"
	"strings"
	"time"
)

func formatNumber(value float64, decimals int) string {
	negative := value < 0
	raw := strconv.FormatFloat(math.Abs(value), 'f', decimals, 64)
	intPart, fracPart, _ := strings.Cut(raw, ".")
	var b strings.Builder
	for i, ch := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(ch)
	}
	out := b.String()
	if fracPart != "" {
		out += "," + fracPart
	}
	if negative && strings.Trim(out, "0.,") != "" {
		out = "-" + out
	}
	return out
}

func formatSignedPct(value float64) string {
	if value > 0 {
		return "+" + formatNumber(value, 1) + "%"
	}
	return formatNumber(value, 1) + "%"
}

func formatPct(value float64) string {
	return formatNumber(math.Abs(value), 1) + "%"
}

func formatKWh(value float64) string {
	return formatNumber(value, 0) + " kWh"
}

func formatTime(t time.Time) string {
	return t.Format("02/01 15:04")
}
