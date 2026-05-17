//go:build linux

package session

import (
	"encoding/json"
	"strings"
)

func encodeRecordReferences(refs []RecordReference) string {
	normalized := NormalizeRecordReferences(refs)
	if len(normalized) == 0 {
		return "[]"
	}
	buf, err := json.Marshal(normalized)
	if err != nil {
		return "[]"
	}
	return string(buf)
}

func decodeRecordReferences(raw string) []RecordReference {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var refs []RecordReference
	if err := json.Unmarshal([]byte(raw), &refs); err != nil {
		return nil
	}
	return NormalizeRecordReferences(refs)
}
