package persistence

import "math"

func int64ToUint32(v int64) (uint32, bool) {
	if v < 0 || v > math.MaxUint32 {
		return 0, false
	}

	//nolint:gosec // Value is bounds-checked above.
	return uint32(v), true
}

func int64ToInt32(v int64) (int32, bool) {
	if v < math.MinInt32 || v > math.MaxInt32 {
		return 0, false
	}

	//nolint:gosec // Value is bounds-checked above.
	return int32(v), true
}
