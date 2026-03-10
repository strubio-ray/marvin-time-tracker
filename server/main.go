package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println("marvin-relay " + version)
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

	history := NewHistoryStore(cfg.HistoryFilePath)
	if err := history.Load(); err != nil {
		log.Fatalf("history load error: %v", err)
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
		log.Printf("APNs client initialized (%s)", apnsEnv)
	} else {
		log.Printf("APNs not configured, push notifications disabled")
	}

	broker := NewBroker()

	marvin := NewMarvinClient(cfg.MarvinAPIToken, cfg.MarvinFullAccessToken)

	// Start 8-hour Live Activity renewal
	renewal := NewRenewal(store, notifier, broker)
	renewal.Start()
	log.Printf("renewal monitor started")

	if cfg.APIKey == "" {
		log.Printf("WARNING: API_KEY not set, app endpoints are unprotected")
	}

	srv := NewServer(store, dedup, notifier, WithBroker(broker), WithMarvinClient(marvin), WithHistory(history), WithExternalURL(cfg.ExternalURL), WithAPIKey(cfg.APIKey))

	log.Printf("listening on %s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, srv); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
