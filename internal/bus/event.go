package bus

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
	"Vortex_Refinery/pkg/types"
)

type EventBus struct {
	client        *redis.Client
	streamKey     string
	consumerGroup string
}

func NewEventBus(addr, password string, db int, streamKey, consumerGroup string) (*EventBus, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	// Create consumer group if not exists (ignore error if already exists)
	_ = client.XGroupCreateMkStream(ctx, streamKey, consumerGroup, "0")

	return &EventBus{
		client:        client,
		streamKey:     streamKey,
		consumerGroup: consumerGroup,
	}, nil
}

func (b *EventBus) PushEvent(ctx context.Context, event *types.Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// Use XAdd to write to Redis Stream
	return b.client.XAdd(ctx, &redis.XAddArgs{
		Stream: b.streamKey,
		ID:     "*",
		Values: map[string]interface{}{
			"data": string(data),
		},
	}).Err()
}

func (b *EventBus) ReadEvents(ctx context.Context, count int64) ([]*types.Event, error) {
	streams, err := b.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    b.consumerGroup,
		Consumer: "master",
		Streams:  []string{b.streamKey, ">"},
		Count:    count,
		Block:    0,
	}).Result()

	if err != nil {
		return nil, err
	}

	var events []*types.Event
	for _, stream := range streams {
		for _, msg := range stream.Messages {
			data, ok := msg.Values["data"].(string)
			if !ok {
				continue
			}
			var event types.Event
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}
			events = append(events, &event)
			// Acknowledge the message
			b.client.XAck(ctx, b.streamKey, b.consumerGroup, msg.ID)
		}
	}

	return events, nil
}

func (b *EventBus) Close() error {
	return b.client.Close()
}
