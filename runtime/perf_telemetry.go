//go:build linux

package runtime

import "time"

func durationMillis(d time.Duration) int64 {
	if d < 0 {
		return 0
	}
	return d.Milliseconds()
}

func elapsedMillisSince(start time.Time) int64 {
	if start.IsZero() {
		return 0
	}
	return durationMillis(time.Since(start))
}

func putDurationMillis(payload map[string]any, key string, d time.Duration) {
	if payload == nil || key == "" {
		return
	}
	payload[key] = durationMillis(d)
}
