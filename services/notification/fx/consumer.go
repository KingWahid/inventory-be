package fx

import (
	"context"
	"sync"

	uberfx "go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/KingWahid/inventory/backend/pkg/alertworker"
	"github.com/KingWahid/inventory/backend/pkg/eventbus"
	"github.com/KingWahid/inventory/backend/services/notification/config"
	"github.com/KingWahid/inventory/backend/services/notification/service"
)

// RegisterStreamConsumer starts Redis Streams consumer group "notification" when configured.
func RegisterStreamConsumer(lc uberfx.Lifecycle, cfg *config.Config, log *zap.Logger, d *service.Dispatcher) {
	if !cfg.StreamConsumerConfigured() {
		switch {
		case !cfg.NotifStreamConsumerEnabled:
			log.Info("notification stream consumer disabled (NOTIF_STREAM_CONSUMER_ENABLED=false)")
		default:
			log.Info("notification stream consumer skipped (need REDIS_ADDR and EVENTBUS_HMAC_SECRET)")
		}
		return
	}

	bus, err := eventbus.New(cfg.RedisAddr)
	if err != nil {
		log.Error("notification stream consumer: eventbus client", zap.Error(err))
		return
	}

	runCtx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	handler := func(ctx context.Context, ev eventbus.BaseEvent) error {
		return d.Handle(ctx, ev)
	}
	awCfg := alertworker.Config{
		ConsumerName: cfg.NotifConsumerName,
		Group:        eventbus.GroupFor("notification"),
	}

	lc.Append(uberfx.Hook{
		OnStart: func(context.Context) error {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := alertworker.Run(runCtx, bus, cfg.EventBusHMACSecret, handler, awCfg); err != nil && err != context.Canceled {
					log.Error("notification stream consumer exited", zap.Error(err))
				}
			}()
			log.Info("notification stream consumer started",
				zap.String("group", awCfg.Group),
				zap.String("consumer", cfg.NotifConsumerName),
			)
			return nil
		},
		OnStop: func(context.Context) error {
			cancel()
			wg.Wait()
			if err := bus.Close(); err != nil {
				log.Warn("notification eventbus close", zap.Error(err))
			}
			return nil
		},
	})
}
