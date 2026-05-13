package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/capyflow/allspark-go/logx"
	"github.com/capyflow/allspark-go/system"

	"Vortex_Refinery/conf"
	"Vortex_Refinery/internal/bus"
	"Vortex_Refinery/internal/master"
	"Vortex_Refinery/internal/store"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = ctx // suppress unused warning (ctx passed indirectly via master)

	cfgPath := flag.String("conf", "config/master.yaml", "path to config file")
	flag.Parse()

	config, err := conf.Load(*cfgPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	logx.Infof("Bootstrap|Config|Loaded|conf=%s port=%d workflows_dir=%s",
		*cfgPath, config.Port, config.Workflows.Dir)

	// Initialize MongoDB store
	mongoURI := config.MongoDB.URI
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	mongoStore, err := store.NewMongoStore(mongoURI, config.MongoDB.Database)
	if err != nil {
		log.Fatalf("failed to connect to mongo: %v", err)
	}
	logx.Infof("Bootstrap|MongoStore|Connected|database=%s", config.MongoDB.Database)

	// Initialize event bus
	eventBus, err := bus.NewEventBus(
		config.Redis.Addr,
		config.Redis.Password,
		config.Redis.DB,
		config.Redis.StreamKey,
		"vortex:masters",
	)
	if err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}

	// Initialize task bus
	taskBus, err := bus.NewTaskBus(
		config.Redis.Addr,
		config.Redis.Password,
		config.Redis.DB,
		config.Redis.TaskStreamKey,
		"vortex:workers",
	)
	if err != nil {
		log.Fatalf("failed to connect to redis task bus: %v", err)
	}

	// Start master
	m := master.New(mongoStore, eventBus, taskBus, config)
	if err := m.Start(); err != nil {
		log.Fatalf("master failed: %v", err)
	}
	logx.Infof("Bootstrap|Master|Started|port=%d", config.Port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logx.Info("Bootstrap|Shutdown|Signal|received")

	shutdownCtx := context.Background()
	if err := m.Shutdown(shutdownCtx); err != nil {
		logx.Errorf("Bootstrap|Shutdown|Error=%v", err)
	}
	if err := mongoStore.Close(shutdownCtx); err != nil {
		logx.Errorf("Bootstrap|MongoStore|Close|Error=%v", err)
	}
	system.GracefulShutdown(func(ctx context.Context) error { return nil })
}
