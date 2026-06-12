package services

import "fmt"

const maxInt32Value = 1<<31 - 1

func int32Size(value int64) (int32, error) {
	if value > maxInt32Value {
		return 0, fmt.Errorf("archive metadata size exceeds int32 limit: %d", value)
	}
	return int32(value), nil //nolint:gosec // value is checked against maxInt32Value above.
}

func int32Len(value int) (int32, error) {
	if value > maxInt32Value {
		return 0, fmt.Errorf("archive metadata size exceeds int32 limit: %d", value)
	}
	return int32(value), nil //nolint:gosec // value is checked against maxInt32Value above.
}
