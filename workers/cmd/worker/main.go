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
	"github.com/KingWahid/inventory/backend/workers/internal/outboxrelay"
)

func main() {
	cfg := viper.New()
	cfg.AutomaticEnv()
	cfg.SetDefault("WORKER_MODE", "all")
	cfg.SetDefault("OUTBOX_RELAY_POLL_MS", 500)
	cfg.SetDefault("OUTBOX_RELAY_BATCH", 100)

	mode := flag.String("mode", cfg.GetString("WORKER_MODE"), "worker mode: all | outbox-relay")
	flag.Parse()

	log.Printf("worker starting mode=%s", *mode)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	var cleanup func()

	if *mode == "all" || *mode == "outbox-relay" {
		dsn := cfg.GetString("DB_DSN")
		redisAddr := cfg.GetString("REDIS_ADDR")
		secret := cfg.GetString("EVENTBUS_HMAC_SECRET")
		if secret == "" {
			log.Fatal("EVENTBUS_HMAC_SECRET is required for outbox relay")
		}
		if dsn == "" {
			log.Fatal("DB_DSN is required for outbox relay")
		}
		if redisAddr == "" {
			log.Fatal("REDIS_ADDR is required for outbox relay")
		}

		sqlDB, err := sql.Open("pgx", dsn)
		if err != nil {
			log.Fatalf("db open: %v", err)
		}
		if err := sqlDB.PingContext(ctx); err != nil {
			_ = sqlDB.Close()
			log.Fatalf("db ping: %v", err)
		}

		gdb, err := postgres.OpenGORM(sqlDB)
		if err != nil {
			_ = sqlDB.Close()
			log.Fatalf("gorm: %v", err)
		}

		bus, err := eventbus.New(redisAddr)
		if err != nil {
			_ = sqlDB.Close()
			log.Fatalf("redis: %v", err)
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
			Bus:    bus,
			Secret: secret,
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

		cleanup = func() {
			_ = bus.Close()
			_ = sqlDB.Close()
		}
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("worker shutting down")
	cancel()
	wg.Wait()
	if cleanup != nil {
		cleanup()
	}
}
