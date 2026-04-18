package requestmeta

import (
	"context"
	"testing"
)

func TestMeta_roundTrip(t *testing.T) {
	t.Parallel()
	ip := "10.0.0.1"
	req := "req-xyz"
	meta := Meta{IP: &ip, RequestID: &req}
	ctx := WithContext(context.Background(), meta)
	got := FromContext(ctx)
	if got.IP == nil || *got.IP != ip {
		t.Fatal("IP lost")
	}
	if got.RequestID == nil || *got.RequestID != req {
		t.Fatal("request id lost")
	}
}
