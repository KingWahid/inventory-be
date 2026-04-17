package eventbus

import (
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
)

type EventMessage struct {
	ID      string
	Values  map[string]any
	Stream  string
	Group   string
	Consumer string
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
