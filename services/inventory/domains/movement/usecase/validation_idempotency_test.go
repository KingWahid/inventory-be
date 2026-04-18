package usecase

import (
	"errors"
	"strings"
	"testing"

	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
)

func TestValidateIdempotencyCreate(t *testing.T) {
	t.Parallel()
	validHash := strings.Repeat("a", 64)

	tests := []struct {
		name    string
		in      CreateMovementBase
		wantErr bool
	}{
		{"empty_key", CreateMovementBase{IdempotencyKey: "", RequestHashSHA256Hex: validHash}, true},
		{"key_too_long", CreateMovementBase{IdempotencyKey: strings.Repeat("x", 256), RequestHashSHA256Hex: validHash}, true},
		{"bad_hash_len", CreateMovementBase{IdempotencyKey: "k1", RequestHashSHA256Hex: "abc"}, true},
		{"bad_hash_char", CreateMovementBase{IdempotencyKey: "k1", RequestHashSHA256Hex: strings.Repeat("g", 64)}, true},
		{"ok", CreateMovementBase{IdempotencyKey: "idem-1", RequestHashSHA256Hex: validHash}, false},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateIdempotencyCreate(tc.in)
			if tc.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected: %v", err)
			}
			if tc.wantErr && err != nil && !errors.Is(err, errorcodes.ErrValidationError) {
				t.Fatalf("want validation error, got %v", err)
			}
		})
	}
}
