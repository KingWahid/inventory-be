package eventbus

import (
	"fmt"
	"strings"
)

const (
	DefaultInventoryStream = "inventory.events"
	DefaultDeadLetter      = "inventory.events.dead"
	DefaultAlertsGroup     = "alerts"
)

func StreamName(domain, entity string, version int) string {
	if version <= 0 {
		version = 1
	}
	return fmt.Sprintf("%s:%s:v%d", safeName(domain, "inventory"), safeName(entity, "events"), version)
}

func EventType(domain, entity, action string) string {
	return fmt.Sprintf("%s.%s.%s", safeName(domain, "inventory"), safeName(entity, "event"), safeName(action, "unknown"))
}

func ConsumerGroup(service, entity string) string {
	return fmt.Sprintf("%s-%s-consumer", safeName(service, "service"), safeName(entity, "events"))
}

func DLQStream(originalStream string) string {
	if strings.TrimSpace(originalStream) == "" {
		return "dlq:" + DefaultInventoryStream
	}
	return "dlq:" + originalStream
}

func StreamInventoryEvents() string {
	// Backward-compatible wrapper.
	return DefaultInventoryStream
}

func StreamDeadLetter() string {
	// Backward-compatible wrapper.
	return DefaultDeadLetter
}

func GroupAlerts() string {
	// Backward-compatible wrapper.
	return DefaultAlertsGroup
}

func GroupFor(name string) string {
	if name == "" {
		return DefaultAlertsGroup
	}
	return name
}

func safeName(v, fallback string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return fallback
	}
	return v
}
