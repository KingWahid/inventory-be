package eventbus

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// SignEvent creates an HMAC-SHA256 signature for the event envelope.
func SignEvent(secret string, event BaseEvent) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", errors.New("eventbus: secret is required")
	}
	canonical, err := canonicalString(event)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(canonical))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// VerifyEvent checks event.Signature against the canonical envelope content.
func VerifyEvent(secret string, event BaseEvent) (bool, error) {
	if event.Signature == "" {
		return false, errors.New("eventbus: signature is required")
	}
	expected, err := SignEvent(secret, event)
	if err != nil {
		return false, err
	}
	return hmac.Equal([]byte(expected), []byte(event.Signature)), nil
}

func canonicalString(event BaseEvent) (string, error) {
	if event.ID == "" {
		return "", errors.New("eventbus: event id is required")
	}
	if event.Type == "" {
		return "", errors.New("eventbus: event type is required")
	}
	if event.Stream == "" {
		return "", errors.New("eventbus: event stream is required")
	}
	if len(event.Payload) == 0 {
		return "", errors.New("eventbus: event payload is required")
	}

	return fmt.Sprintf("%s|%s|%d|%s|%s|%s|%s",
		event.ID,
		event.Type,
		event.Version,
		event.Stream,
		event.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
		event.PublishedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
		string(event.Payload),
	), nil
}
