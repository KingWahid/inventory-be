package httpresponse

import "testing"

func TestComputeTotalPages(t *testing.T) {
	tests := []struct {
		total    int64
		perPage  int64
		wantPages int32
	}{
		{150, 20, 8},
		{0, 20, 0},
		{1, 20, 1},
		{20, 20, 1},
		{21, 20, 2},
		{100, 0, 0},
	}
	for _, tt := range tests {
		if got := ComputeTotalPages(tt.total, tt.perPage); got != tt.wantPages {
			t.Fatalf("ComputeTotalPages(%d,%d)=%d want %d", tt.total, tt.perPage, got, tt.wantPages)
		}
	}
}
