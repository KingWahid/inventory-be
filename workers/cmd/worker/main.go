package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/spf13/viper"

	"github.com/KingWahid/inventory/backend/infra/postgres"
	"github.com/KingWahid/inventory/backend/pkg/eventbus"
	outboxrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/outbox/repository"
	"github.com/KingWahid/inventory/backend/workers/internal/alertworker"
	"github.com/KingWahid/inventory/backend/workers/internal/outboxrelay"
)

func main() {
	cfg := viper.New()
	cfg.AutomaticEnv()
	cfg.SetDefault("WORKER_MODE", "all")
	cfg.SetDefault("OUTBOX_RELAY_POLL_MS", 500)
	cfg.SetDefault("OUTBOX_RELAY_BATCH", 100)
	cfg.SetDefault("ALERT_CONSUMER_NAME", "worker-alerts-1")

	mode := flag.String("mode", cfg.GetString("WORKER_MODE"), "worker mode: all | outbox-relay | alerts")
	flag.Parse()

	log.Printf("worker starting mode=%s", *mode)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	var cleanups []func()

	switch *mode {
	case "all", "outbox-relay", "alerts":
	default:
		log.Printf("worker mode=%s: no background tasks (waiting for shutdown)", *mode)
	}

	redisAddr := cfg.GetString("REDIS_ADDR")
	eventSecret := cfg.GetString("EVENTBUS_HMAC_SECRET")

	var sharedBus *eventbus.Client
	switch *mode {
	case "all", "outbox-relay", "alerts":
		if redisAddr == "" {
			log.Fatal("REDIS_ADDR is required for this worker mode")
		}
		bus, err := eventbus.New(redisAddr)
		if err != nil {
			log.Fatalf("redis: %v", err)
		}
		sharedBus = bus
		cleanups = append(cleanups, func() { _ = bus.Close() })
	}

	if *mode == "all" || *mode == "outbox-relay" {
		if eventSecret == "" {
			log.Fatal("EVENTBUS_HMAC_SECRET is required for outbox relay")
		}
		dsn := cfg.GetString("DB_DSN")
		if dsn == "" {
			log.Fatal("DB_DSN is required for outbox relay")
		}

		sqlDB, err := sql.Open("pgx", dsn)
		if err != nil {
			log.Fatalf("db open: %v", err)
		}
		cleanups = append(cleanups, func() { _ = sqlDB.Close() })

		if err := sqlDB.PingContext(ctx); err != nil {
			log.Fatalf("db ping: %v", err)
		}

		gdb, err := postgres.OpenGORM(sqlDB)
		if err != nil {
			log.Fatalf("gorm: %v", err)
		}

		repo := outboxrepo.New(gdb)
		pollMs := cfg.GetInt("OUTBOX_RELAY_POLL_MS")
		if pollMs <= 0 {
			pollMs = 500
		}
		batch := cfg.GetInt("OUTBOX_RELAY_BATCH")
		if batch <= 0 {
			batch = 100
		}

		runner := &outboxrelay.Runner{
			Repo:   repo,
			Bus:    sharedBus,
			Secret: eventSecret,
			Config: outboxrelay.Config{
				PollInterval: time.Duration(pollMs) * time.Millisecond,
				BatchSize:    batch,
			},
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := runner.Run(ctx); err != nil && err != context.Canceled {
				log.Printf("outbox relay stopped: %v", err)
			}
		}()
	}

	if *mode == "all" || *mode == "alerts" {
		if eventSecret == "" {
			log.Fatal("EVENTBUS_HMAC_SECRET is required for alert worker (HMAC verify)")
		}
		if sharedBus == nil {
			log.Fatal("internal error: redis client missing for alerts")
		}
		conName := cfg.GetString("ALERT_CONSUMER_NAME")

		wg.Add(1)
		go func() {
			defer wg.Done()
			err := alertworker.Run(ctx, sharedBus, eventSecret, alertworker.StubHandler, alertworker.Config{
				ConsumerName: conName,
			})
			if err != nil && err != context.Canceled {
				log.Printf("alert worker stopped: %v", err)
			}
		}()
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("worker shutting down")
	cancel()
	wg.Wait()
	for i := len(cleanups) - 1; i >= 0; i-- {
		cleanups[i]()
	}
}
