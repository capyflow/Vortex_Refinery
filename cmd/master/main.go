package main

import (
	"flag"
	"log"

	"Vortex_Refinery/config"
	"Vortex_Refinery/internal/bus"
	"Vortex_Refinery/internal/master"
	"Vortex_Refinery/internal/store"
)

func main() {
	configPath := flag.String("config", "config/master.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize MongoDB store
	mongoStore, err := store.NewMongoStore(cfg.MongoDB.URI, cfg.MongoDB.Database)
	if err != nil {
		log.Fatalf("failed to connect to mongo: %v", err)
	}

	// Initialize event bus
	eventBus, err := bus.NewEventBus(
		cfg.Redis.Addr,
		cfg.Redis.Password,
		cfg.Redis.DB,
		cfg.Redis.StreamKey,
		"vortex:masters",
	)
	if err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}

	// Initialize task bus
	taskBus, err := bus.NewTaskBus(
		cfg.Redis.Addr,
		cfg.Redis.Password,
		cfg.Redis.DB,
		cfg.Redis.TaskStreamKey,
		"vortex:workers",
	)
	if err != nil {
		log.Fatalf("failed to connect to redis task bus: %v", err)
	}

	// Start master
	m := master.New(mongoStore, eventBus, taskBus, cfg)
	if err := m.Start(); err != nil {
		log.Fatalf("master failed: %v", err)
	}
}
