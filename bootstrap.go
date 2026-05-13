package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/capyflow/allspark-go/logx"
	"github.com/capyflow/allspark-go/system"

	"Vortex_Refinery/conf"
	"Vortex_Refinery/internal/store"
	"Vortex_Refinery/server"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfgPath := flag.String("conf", "config/master.yaml", "config file path")
	flag.Parse()

	// Load config using the existing Load function
	config, err := conf.Load(*cfgPath)
	if err != nil {
		fmt.Printf("failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Default port from server.http_addr if not set
	if config.Port == 0 {
		config.Port = 8088
	}
	if config.Workflows.Dir == "" {
		config.Workflows.Dir = "./workflows"
	}
	if config.MongoDB.Database == "" {
		config.MongoDB.Database = "vortex_refinery"
	}

	logx.Infof("Bootstrap|Config|Loaded|conf=%s port=%d workflows_dir=%s", *cfgPath, config.Port, config.Workflows.Dir)

	// Build DBConfig for allspark
	config.BuildDBConfig()

	// allspark DatabaseServer initialization skipped (dsServer not used by refinery)
	logx.Info("Bootstrap|DS|Skipped")

	// Initialize MongoDB store
	mongoHost := "localhost"
	mongoPort := 27017
	if config.DBConfig != nil && config.DBConfig.Mongo != nil {
		mongoHost = config.DBConfig.Mongo.Host
		mongoPort = config.DBConfig.Mongo.Port
	}
	mongoURI := fmt.Sprintf("mongodb://%s:%d", mongoHost, mongoPort)
	if config.MongoDB.URI != "" {
		mongoURI = config.MongoDB.URI
	}

	mongoStore, err := store.NewMongoStore(mongoURI, config.MongoDB.Database)
	if err != nil {
		logx.Errorf("Bootstrap|MongoStore|Init|Error=%v", err)
		os.Exit(1)
	}
	logx.Infof("Bootstrap|MongoStore|Connected|database=%s", config.MongoDB.Database)

	// Start Refinery HTTP Server
	refineryServer := server.NewRefineryServer(ctx, config, mongoStore)
	go refineryServer.Start()

	logx.Infof("Bootstrap|Refinery|Started|port=%d", config.Port)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logx.Info("Bootstrap|Shutdown|Signal|received")

	shutdownCtx := context.Background()
	if err := refineryServer.Shutdown(shutdownCtx); err != nil {
		logx.Errorf("Bootstrap|Shutdown|Error=%v", err)
	}
	if err := mongoStore.Close(shutdownCtx); err != nil {
		logx.Errorf("Bootstrap|MongoStore|Close|Error=%v", err)
	}
	system.GracefulShutdown(func(ctx context.Context) error { return nil })
}
