package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println("marvin-relay 0.1.0")
		os.Exit(0)
	}

	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	store := NewStateStore(cfg.StateFilePath)
	if err := store.Load(); err != nil {
		log.Fatalf("state load error: %v", err)
	}

	dedup := NewDedupCache(60 * time.Second)

	// Initialize APNs notifier if configured
	var notifier Notifier
	if cfg.APNsKeyID != "" && cfg.APNsTeamID != "" && cfg.APNsPrivateKeyPath != "" {
		apnsClient, err := NewAPNsClient(cfg.APNsPrivateKeyPath, cfg.APNsKeyID, cfg.APNsTeamID, cfg.APNsBundleID)
		if err != nil {
			log.Fatalf("APNs init error: %v", err)
		}
		notifier = apnsClient
		log.Printf("APNs client initialized")
	} else {
		log.Printf("APNs not configured, push notifications disabled")
	}

	// Initialize Marvin client and poller
	marvin := NewMarvinClient(cfg.MarvinAPIToken)
	if cfg.PollEnabled {
		quota := NewQuotaCounter()
		poller := NewPoller(marvin, store, notifier, cfg.PollIntervalActive, cfg.PollIntervalIdle, quota)
		poller.Start()
		log.Printf("poller started (active=%v, idle=%v)", cfg.PollIntervalActive, cfg.PollIntervalIdle)
	} else {
		log.Printf("poller disabled (webhooks only)")
	}

	// Start 8-hour Live Activity renewal
	renewal := NewRenewal(store, notifier)
	renewal.Start()
	log.Printf("renewal monitor started")

	srv := NewServer(store, dedup, notifier, WithMarvinClient(marvin))

	log.Printf("listening on %s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, srv); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
