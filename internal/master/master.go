package master

import (
	"log"

	"Vortex_Refinery/config"
	"Vortex_Refinery/internal/bus"
	"Vortex_Refinery/internal/store"
)

type Master struct {
	cfg       *config.Config
	store     *store.MongoStore
	eventBus  *bus.EventBus
	taskBus   *bus.TaskBus
}

func New(s *store.MongoStore, eb *bus.EventBus, tb *bus.TaskBus, cfg *config.Config) *Master {
	return &Master{
		cfg:       cfg,
		store:     s,
		eventBus:  eb,
		taskBus:   tb,
	}
}

func (m *Master) Start() error {
	log.Println("Master starting...")

	// TODO: Initialize gRPC server
	// TODO: Start event consumer loop
	// TODO: Start task result receiver
	// TODO: Start heartbeat checker

	log.Println("Master started successfully")
	return nil
}
