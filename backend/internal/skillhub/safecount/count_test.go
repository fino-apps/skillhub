package safecount

import (
	"math"
	"testing"
)

func TestInt32(t *testing.T) {
	for _, tc := range []struct {
		input int64
		want  int32
	}{
		{-1, 0}, {0, 0}, {42, 42}, {math.MaxInt32, math.MaxInt32}, {math.MaxInt32 + 1, math.MaxInt32},
	} {
		if got := Int32(tc.input); got != tc.want {
			t.Errorf("Int32(%d) = %d, want %d", tc.input, got, tc.want)
		}
	}
}
