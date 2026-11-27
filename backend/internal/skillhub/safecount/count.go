// Package safecount converts non-negative counts to the API's int32 range.
package safecount

import "math"

func Int32(n int64) int32 {
	if n > math.MaxInt32 {
		return math.MaxInt32
	}
	if n < 0 {
		return 0
	}
	return int32(n) // #nosec G115 -- n is within the int32 range.
}
