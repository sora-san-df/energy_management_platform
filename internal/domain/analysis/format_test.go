package analysis

import (
	"testing"
	"time"
)

func TestFormatNumberUsesSpanishSeparators(t *testing.T) {
	cases := []struct {
		value    float64
		decimals int
		want     string
	}{
		{2180, 0, "2.180"},
		{1234567.891, 2, "1.234.567,89"},
		{103.66, 1, "103,7"},
		{-5.26, 1, "-5,3"},
		{-0.01, 1, "0,0"},
		{0, 0, "0"},
	}
	for _, c := range cases {
		if got := formatNumber(c.value, c.decimals); got != c.want {
			t.Errorf("formatNumber(%v, %d) = %q, want %q", c.value, c.decimals, got, c.want)
		}
	}
}

func TestFormatSignedPct(t *testing.T) {
	if got := formatSignedPct(103.7); got != "+103,7%" {
		t.Fatalf("got %q", got)
	}
	if got := formatSignedPct(-1.4); got != "-1,4%" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatTime(t *testing.T) {
	if got := formatTime(time.Date(2026, 9, 12, 14, 0, 0, 0, time.UTC)); got != "12/09 14:00" {
		t.Fatalf("got %q", got)
	}
}
