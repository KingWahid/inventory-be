package eventbus

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func TestStreamHelpers(t *testing.T) {
	if StreamInventoryEvents() != "inventory.events" {
		t.Fatalf("unexpected inventory stream: %s", StreamInventoryEvents())
	}
	if StreamDeadLetter() != "inventory.events.dead" {
		t.Fatalf("unexpected dead-letter stream: %s", StreamDeadLetter())
	}
	if GroupAlerts() != "alerts" {
		t.Fatalf("unexpected alerts group: %s", GroupAlerts())
	}
}

func TestGroupForDefault(t *testing.T) {
	if GroupFor("") != "alerts" {
		t.Fatalf("expected default group alerts, got %s", GroupFor(""))
	}
	if GroupFor("notifications") != "notifications" {
		t.Fatalf("expected group notifications, got %s", GroupFor("notifications"))
	}
}

func TestRetryCount(t *testing.T) {
	cases := []struct {
		name   string
		values map[string]any
		want   int
	}{
		{name: "missing", values: map[string]any{}, want: 0},
		{name: "int", values: map[string]any{"retry_count": 2}, want: 2},
		{name: "string", values: map[string]any{"retry_count": "3"}, want: 3},
		{name: "float", values: map[string]any{"retry_count": 4.0}, want: 4},
		{name: "invalid", values: map[string]any{"retry_count": "x"}, want: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RetryCount(tc.values); got != tc.want {
				t.Fatalf("RetryCount() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestWithRetryMetadata(t *testing.T) {
	in := map[string]any{"event_type": "StockChanged"}
	out := WithRetryMetadata(in, 2, errors.New("processing failed"))

	if out["event_type"] != "StockChanged" {
		t.Fatalf("expected original payload field")
	}
	if out["retry_count"] != 2 {
		t.Fatalf("expected retry_count=2, got %v", out["retry_count"])
	}
	if out["last_error"] != "processing failed" {
		t.Fatalf("expected last_error, got %v", out["last_error"])
	}
	if _, ok := in["retry_count"]; ok {
		t.Fatalf("input map should not be mutated")
	}
}

func TestErrorClassification(t *testing.T) {
	if !IsTransient(Transient(errors.New("timeout"))) {
		t.Fatal("expected transient error classification")
	}
	if IsPermanent(Transient(errors.New("timeout"))) {
		t.Fatal("transient error should not be permanent")
	}

	if !IsPermanent(Permanent(errors.New("bad payload"))) {
		t.Fatal("expected permanent error classification")
	}
}

func TestBuildIdempotencyKey(t *testing.T) {
	msg := EventMessage{
		ID:     "168-0",
		Stream: "inventory.events",
		Group:  "alerts",
	}
	got := BuildIdempotencyKey(msg)
	want := "eventbus:idempotency:inventory.events:alerts:168-0"
	if got != want {
		t.Fatalf("unexpected idempotency key: got %s want %s", got, want)
	}
}

func TestBuildIdempotencyKeyPrefersEventID(t *testing.T) {
	payload, err := json.Marshal(map[string]any{"foo": "bar"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	msg := EventMessage{
		ID:     "redis-id-1",
		Stream: "inventory.events",
		Group:  "alerts",
		Values: map[string]any{
			FieldEventID:          "evt-123",
			FieldEventType:        "inventory.stock.changed",
			FieldEventVersion:     "1",
			FieldEventStream:      "inventory.events",
			FieldEventCreatedAt:   time.Now().UTC().Format(time.RFC3339Nano),
			FieldEventPublishedAt: time.Now().UTC().Format(time.RFC3339Nano),
			FieldEventPayload:     string(payload),
		},
	}
	got := BuildIdempotencyKey(msg)
	want := "eventbus:idempotency:inventory.events:alerts:evt-123"
	if got != want {
		t.Fatalf("unexpected idempotency key: got %s want %s", got, want)
	}
}

func TestPublishEventAndDecodeEvent(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(func() { mr.Close() })

	c, err := New(mr.Addr())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	ctx := context.Background()
	stream := StreamName("inventory", "stock", 1)
	group := ConsumerGroup("inventory", "stock")
	consumer := "c1"
	if err := c.EnsureGroup(ctx, stream, group); err != nil {
		t.Fatalf("EnsureGroup: %v", err)
	}

	payload, err := EncodeEventPayload(map[string]any{"sku": "A-1"})
	if err != nil {
		t.Fatalf("EncodeEventPayload: %v", err)
	}
	ev := BaseEvent{
		ID:          "evt-1",
		Type:        EventType("inventory", "stock", "changed"),
		Version:     1,
		Stream:      stream,
		CreatedAt:   time.Now().UTC(),
		PublishedAt: time.Now().UTC(),
		Payload:     payload,
	}
	if _, err := c.PublishEvent(ctx, ev); err != nil {
		t.Fatalf("PublishEvent: %v", err)
	}

	msgs, err := c.ReadGroup(ctx, stream, group, consumer, 1, time.Millisecond)
	if err != nil {
		t.Fatalf("ReadGroup: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	got, err := DecodeEvent(msgs[0])
	if err != nil {
		t.Fatalf("DecodeEvent: %v", err)
	}
	if got.ID != ev.ID || got.Type != ev.Type || got.Stream != ev.Stream {
		t.Fatalf("decoded event mismatch: %+v", got)
	}
}

func TestSignAndVerifyEvent(t *testing.T) {
	payload, err := EncodeEventPayload(map[string]any{"x": 1})
	if err != nil {
		t.Fatalf("EncodeEventPayload: %v", err)
	}
	ev := BaseEvent{
		ID:          "evt-2",
		Type:        "inventory.stock.changed",
		Version:     1,
		Stream:      "inventory:stock:v1",
		CreatedAt:   time.Now().UTC(),
		PublishedAt: time.Now().UTC(),
		Payload:     payload,
	}
	sig, err := SignEvent("secret", ev)
	if err != nil {
		t.Fatalf("SignEvent: %v", err)
	}
	ev.Signature = sig
	ok, err := VerifyEvent("secret", ev)
	if err != nil {
		t.Fatalf("VerifyEvent: %v", err)
	}
	if !ok {
		t.Fatal("expected signature verification success")
	}

	ev.Payload = []byte(`{"x":2}`)
	ok, err = VerifyEvent("secret", ev)
	if err != nil {
		t.Fatalf("VerifyEvent(tampered): %v", err)
	}
	if ok {
		t.Fatal("expected signature verification failure for tampered payload")
	}
}

func TestNamingConventions(t *testing.T) {
	if got := StreamName("notification", "email", 1); got != "notification:email:v1" {
		t.Fatalf("unexpected stream name: %s", got)
	}
	if got := EventType("notification", "email", "requested"); got != "notification.email.requested" {
		t.Fatalf("unexpected event type: %s", got)
	}
	if got := ConsumerGroup("notification", "email"); got != "notification-email-consumer" {
		t.Fatalf("unexpected consumer group: %s", got)
	}
	if got := DLQStream("notification:email:v1"); got != "dlq:notification:email:v1" {
		t.Fatalf("unexpected dlq stream: %s", got)
	}
}

func TestHandleMessage_IdempotencySkipsHandler(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(func() { mr.Close() })

	c, err := New(mr.Addr())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	ctx := context.Background()
	stream := StreamInventoryEvents()
	group := GroupAlerts()
	consumer := "c1"

	if err := c.EnsureGroup(ctx, stream, group); err != nil {
		t.Fatalf("EnsureGroup: %v", err)
	}

	publishRead := func(payload map[string]any) EventMessage {
		t.Helper()
		if _, err := c.Publish(ctx, stream, payload); err != nil {
			t.Fatalf("Publish: %v", err)
		}
		msgs, err := c.ReadGroup(ctx, stream, group, consumer, 1, time.Millisecond)
		if err != nil {
			t.Fatalf("ReadGroup: %v", err)
		}
		if len(msgs) != 1 {
			t.Fatalf("expected 1 message, got %d", len(msgs))
		}
		return msgs[0]
	}

	var runs int
	handler := func(ctx context.Context, msg EventMessage) error {
		runs++
		return nil
	}

	msg1 := publishRead(map[string]any{"k": "v"})
	res1, err := c.HandleMessage(ctx, msg1, HandleOptions{
		IdempotencyKey: "idem:outbox:1",
		IdempotencyTTL: time.Minute,
		MaxRetry:       3,
	}, handler)
	if err != nil {
		t.Fatalf("HandleMessage #1: %v", err)
	}
	if !res1.Acked || res1.Skipped {
		t.Fatalf("unexpected res1: %+v", res1)
	}
	if runs != 1 {
		t.Fatalf("expected handler runs=1, got %d", runs)
	}

	msg2 := publishRead(map[string]any{"k": "v"})
	res2, err := c.HandleMessage(ctx, msg2, HandleOptions{
		IdempotencyKey: "idem:outbox:1",
		IdempotencyTTL: time.Minute,
		MaxRetry:       3,
	}, handler)
	if err != nil {
		t.Fatalf("HandleMessage #2: %v", err)
	}
	if !res2.Acked || !res2.Skipped {
		t.Fatalf("expected duplicate skip+ack, got %+v", res2)
	}
	if runs != 1 {
		t.Fatalf("expected handler still runs=1 after duplicate, got %d", runs)
	}
}
