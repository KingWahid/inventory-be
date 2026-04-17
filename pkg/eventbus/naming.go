package eventbus

const (
	DefaultInventoryStream = "inventory.events"
	DefaultDeadLetter      = "inventory.events.dead"
	DefaultAlertsGroup     = "alerts"
)

func StreamInventoryEvents() string {
	return DefaultInventoryStream
}

func StreamDeadLetter() string {
	return DefaultDeadLetter
}

func GroupAlerts() string {
	return DefaultAlertsGroup
}

func GroupFor(name string) string {
	if name == "" {
		return DefaultAlertsGroup
	}
	return name
}
