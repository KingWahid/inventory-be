package usecase

import (
	"encoding/json"
)

func toAuditMap(v any) map[string]any {
	b, err := json.Marshal(v)
	if err != nil {
		return map[string]any{"marshal_error": err.Error()}
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return map[string]any{"unmarshal_error": err.Error()}
	}
	return m
}
