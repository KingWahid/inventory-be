package eventbus

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	FieldEventID          = "event_id"
	FieldEventType        = "event_type"
	FieldEventVersion     = "event_version"
	FieldEventStream      = "event_stream"
	FieldEventCreatedAt   = "event_created_at"
	FieldEventPublishedAt = "event_published_at"
	FieldEventPayload     = "event_payload"
	FieldEventSignature   = "event_signature"
)

type EventMessage struct {
	ID       string
	Values   map[string]any
	Stream   string
	Group    string
	Consumer string
}

// BaseEvent is the canonical event envelope for Redis Streams payload.
type BaseEvent struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Version     int       `json:"version"`
	Stream      string    `json:"stream"`
	CreatedAt   time.Time `json:"created_at"`
	PublishedAt time.Time `json:"published_at"`
	Payload     []byte    `json:"payload"`
	Signature   string    `json:"signature"`
}

func messageFromRedis(stream, group, consumer string, msg redis.XMessage) EventMessage {
	return EventMessage{
		ID:       msg.ID,
		Values:   msg.Values,
		Stream:   stream,
		Group:    group,
		Consumer: consumer,
	}
}

func EncodeEventPayload(v any) ([]byte, error) {
	if v == nil {
		return nil, errors.New("eventbus: payload is required")
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func DecodeEventPayload[T any](payload []byte) (T, error) {
	var out T
	if len(payload) == 0 {
		return out, errors.New("eventbus: payload is empty")
	}
	if err := json.Unmarshal(payload, &out); err != nil {
		return out, err
	}
	return out, nil
}

func (e BaseEvent) ToValues() (map[string]any, error) {
	if e.ID == "" {
		return nil, errors.New("eventbus: event id is required")
	}
	if e.Type == "" {
		return nil, errors.New("eventbus: event type is required")
	}
	if e.Stream == "" {
		return nil, errors.New("eventbus: event stream is required")
	}
	if len(e.Payload) == 0 {
		return nil, errors.New("eventbus: event payload is required")
	}
	if e.Version <= 0 {
		e.Version = 1
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	if e.PublishedAt.IsZero() {
		e.PublishedAt = e.CreatedAt
	}

	return map[string]any{
		FieldEventID:          e.ID,
		FieldEventType:        e.Type,
		FieldEventVersion:     e.Version,
		FieldEventStream:      e.Stream,
		FieldEventCreatedAt:   e.CreatedAt.Format(time.RFC3339Nano),
		FieldEventPublishedAt: e.PublishedAt.Format(time.RFC3339Nano),
		FieldEventPayload:     string(e.Payload),
		FieldEventSignature:   e.Signature,
	}, nil
}

func DecodeEvent(msg EventMessage) (BaseEvent, error) {
	vals := msg.Values
	rawPayload, ok := vals[FieldEventPayload]
	if !ok {
		return BaseEvent{}, errors.New("eventbus: missing event payload")
	}

	version := RetryCount(map[string]any{FieldEventVersion: vals[FieldEventVersion]})
	if version <= 0 {
		version = 1
	}

	createdAt, err := parseTime(anyToString(vals[FieldEventCreatedAt]))
	if err != nil {
		return BaseEvent{}, fmt.Errorf("eventbus: parse created_at: %w", err)
	}
	publishedAt, err := parseTime(anyToString(vals[FieldEventPublishedAt]))
	if err != nil {
		return BaseEvent{}, fmt.Errorf("eventbus: parse published_at: %w", err)
	}

	return BaseEvent{
		ID:          anyToString(vals[FieldEventID]),
		Type:        anyToString(vals[FieldEventType]),
		Version:     version,
		Stream:      anyToString(vals[FieldEventStream]),
		CreatedAt:   createdAt,
		PublishedAt: publishedAt,
		Payload:     []byte(anyToString(rawPayload)),
		Signature:   anyToString(vals[FieldEventSignature]),
	}, nil
}

func parseTime(v string) (time.Time, error) {
	if v == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339Nano, v)
}

func anyToString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case []byte:
		return string(x)
	default:
		return fmt.Sprintf("%v", x)
	}
}

func RetryCount(values map[string]any) int {
	raw, ok := values["retry_count"]
	if !ok {
		return 0
	}
	switch v := raw.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		n, err := strconv.Atoi(v)
		if err != nil {
			return 0
		}
		return n
	default:
		n, err := strconv.Atoi(fmt.Sprintf("%v", v))
		if err != nil {
			return 0
		}
		return n
	}
}
