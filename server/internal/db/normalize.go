package db

import (
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// normalize converts driver values into JSON-encodable ones: dates as
// YYYY-MM-DD, whole numerics as integers, other numerics as floats.
func normalize(v any) any {
	switch x := v.(type) {
	case nil:
		return nil
	case time.Time:
		if x.Hour() == 0 && x.Minute() == 0 && x.Second() == 0 && x.Nanosecond() == 0 {
			return x.Format("2006-01-02")
		}
		return x.Format(time.RFC3339)
	case pgtype.Numeric:
		return numeric(x)
	case pgtype.Time:
		if !x.Valid {
			return nil
		}
		s := x.Microseconds / 1_000_000
		return fmt.Sprintf("%02d:%02d:%02d", s/3600, s/60%60, s%60)
	case float64:
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return nil
		}
		return x
	case float32:
		return normalize(float64(x))
	case [16]byte:
		return fmt.Sprintf("%x-%x-%x-%x-%x", x[0:4], x[4:6], x[6:8], x[8:10], x[10:16])
	case []byte:
		return string(x)
	case string, bool, int, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
		return x
	default:
		return fmt.Sprint(x)
	}
}

func numeric(n pgtype.Numeric) any {
	if !n.Valid || n.NaN || n.InfinityModifier != pgtype.Finite || n.Int == nil {
		return nil
	}
	if n.Exp >= 0 {
		whole := new(big.Int).Mul(n.Int, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n.Exp)), nil))
		if whole.IsInt64() {
			return whole.Int64()
		}
	}
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return nil
	}
	return f.Float64
}
