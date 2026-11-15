package db

import (
	"encoding/json"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestNormalize(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want string // JSON
	}{
		{"nil", nil, "null"},
		{"date", time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), `"2025-06-01"`},
		{"timestamp", time.Date(2025, 6, 1, 9, 5, 0, 0, time.UTC), `"2025-06-01T09:05:00Z"`},
		{"whole numeric", pgtype.Numeric{Int: big.NewInt(2235350040791), Valid: true}, "2235350040791"},
		{"numeric with exponent", pgtype.Numeric{Int: big.NewInt(12), Exp: 3, Valid: true}, "12000"},
		{"fractional numeric", pgtype.Numeric{Int: big.NewInt(1415), Exp: -3, Valid: true}, "1.415"},
		{"null numeric", pgtype.Numeric{}, "null"},
		{"nan numeric", pgtype.Numeric{NaN: true, Valid: true}, "null"},
		{"nan float", math.NaN(), "null"},
		{"time of day", pgtype.Time{Microseconds: (9*3600 + 5*60) * 1_000_000, Valid: true}, `"09:05:00"`},
		{"bytes", []byte("abc"), `"abc"`},
		{"bool", true, "true"},
		{"int64", int64(42), "42"},
		{"text", "normal", `"normal"`},
	}
	for _, c := range cases {
		got, err := json.Marshal(normalize(c.in))
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if string(got) != c.want {
			t.Errorf("%s: got %s want %s", c.name, got, c.want)
		}
	}
}
