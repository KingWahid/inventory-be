package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/spf13/viper"

	"github.com/your-org/inventory/backend/infra/database/cmd/seed/service"
)

func main() {
	mode := flag.String("mode", "seed", "seed mode: seed|rollback")
	flag.Parse()

	cfg := viper.New()
	cfg.AutomaticEnv()

	dsn := cfg.GetString("DB_DSN")
	if dsn == "" {
		log.Fatal("DB_DSN is required")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	seedSvc := service.NewSeedService(db)
	switch *mode {
	case "seed":
		if err := seedSvc.Run(ctx); err != nil {
			log.Fatalf("seed failed: %v", err)
		}
		log.Println("seed completed")
	case "rollback":
		if err := seedSvc.Rollback(ctx); err != nil {
			log.Fatalf("rollback failed: %v", err)
		}
		log.Println("rollback completed")
	default:
		log.Fatalf("unsupported mode: %s", *mode)
	}
}
