package usecase

import (
	"testing"

	movrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/movement/repository"
)

func TestValidateMovementWarehouses(t *testing.T) {
	t.Parallel()
	src := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	dst := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"

	tests := []struct {
		name    string
		typ     string
		src     *string
		dst     *string
		wantErr bool
	}{
		{"inbound_ok", movrepo.TypeInbound, nil, &dst, false},
		{"inbound_bad_has_src", movrepo.TypeInbound, &src, &dst, true},
		{"outbound_ok", movrepo.TypeOutbound, &src, nil, false},
		{"outbound_bad_has_dst", movrepo.TypeOutbound, &src, &dst, true},
		{"transfer_ok", movrepo.TypeTransfer, &src, &dst, false},
		{"transfer_same_wh", movrepo.TypeTransfer, &src, &src, true},
		{"adjustment_inc", movrepo.TypeAdjustment, nil, &dst, false},
		{"adjustment_dec", movrepo.TypeAdjustment, &src, nil, false},
		{"adjustment_both", movrepo.TypeAdjustment, &src, &dst, true},
		{"adjustment_none", movrepo.TypeAdjustment, nil, nil, true},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateMovementWarehouses(tc.typ, tc.src, tc.dst)
			if tc.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected: %v", err)
			}
		})
	}
}
