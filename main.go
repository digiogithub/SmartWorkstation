package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/digiogithub/smartworkstation/internal/config"
	"github.com/digiogithub/smartworkstation/internal/stats"
	"github.com/digiogithub/smartworkstation/internal/tuya"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmsgprefix)

	cfgPath := "config.toml"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Periodic system stats (workstation mode only).
	if cfg.Workstation.Enabled {
		collector := stats.NewCollector(cfg)
		go collector.Start(ctx)
	}

	// Tuya cloud listener + WoL trigger.
	client, err := tuya.NewClient(cfg)
	if err != nil {
		log.Fatalf("tuya: %v", err)
	}
	if err := client.Start(ctx); err != nil {
		log.Fatalf("tuya start: %v", err)
	}

	log.Println("SmartWorkstation running — press Ctrl-C to exit")

	// Block until SIGINT or SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down…")
	cancel()
}
