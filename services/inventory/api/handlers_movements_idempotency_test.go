package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KingWahid/inventory/backend/pkg/idempotency"
	"github.com/KingWahid/inventory/backend/services/inventory/stub"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func TestReadBindRawBodyMovementHash_matchesRawBytes(t *testing.T) {
	t.Parallel()
	wid := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	pid := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	raw := `{"reference_number":"REF-H","destination_warehouse_id":"` + wid.String() + `","lines":[{"product_id":"` + pid.String() + `","quantity":1}]}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(raw))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)

	var body stub.InboundMovementCreateRequest
	got, err := readBindRawBodyMovementHash(c, &body)
	if err != nil {
		t.Fatal(err)
	}
	want := idempotency.SHA256Hex([]byte(raw))
	if got != want {
		t.Fatalf("hash mismatch: got %s want %s", got, want)
	}
	if body.ReferenceNumber != "REF-H" {
		t.Fatalf("bind lost field")
	}
}
