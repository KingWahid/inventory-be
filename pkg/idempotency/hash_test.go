package idempotency

import (
	"strings"
	"testing"
)

func TestSHA256Hex_stable(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"reference_number":"R1","lines":[]}`)
	a := SHA256Hex(raw)
	b := SHA256Hex(raw)
	if a != b {
		t.Fatalf("same bytes must yield same hash")
	}
	if len(a) != 64 {
		t.Fatalf("want 64 hex chars, got %d", len(a))
	}
	if strings.ToLower(a) != a {
		t.Fatalf("hex must be lowercase")
	}
}
