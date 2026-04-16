package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/viper"
)

func main() {
	cfg := viper.New()
	cfg.AutomaticEnv()
	cfg.SetDefault("WORKER_MODE", "all")

	mode := flag.String("mode", cfg.GetString("WORKER_MODE"), "worker mode (stub): all")
	flag.Parse()

	log.Printf("worker starting mode=%s", *mode)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("worker shutting down")
}
