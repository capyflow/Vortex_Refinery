package bus

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
	"Vortex_Refinery/pkg/types"
)

type TaskBus struct {
	client        *redis.Client
	streamKey     string
	consumerGroup string
}

func NewTaskBus(addr, password string, db int, streamKey, consumerGroup string) (*TaskBus, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	// Create consumer group for workers (ignore error if already exists)
	_ = client.XGroupCreateMkStream(ctx, streamKey, consumerGroup, "0")

	return &TaskBus{
		client:        client,
		streamKey:     streamKey,
		consumerGroup: consumerGroup,
	}, nil
}

// DispatchTask dispatches a task to a worker via Redis Stream
func (b *TaskBus) DispatchTask(ctx context.Context, workerID string, task *types.Task) error {
	data, err := json.Marshal(task)
	if err != nil {
		return err
	}

	// XAdd creates a new stream entry with auto-generated ID
	// The worker field is used for routing
	return b.client.XAdd(ctx, &redis.XAddArgs{
		Stream: b.streamKey,
		ID:     "*",
		Values: map[string]interface{}{
			"task":   string(data),
			"worker": workerID,
		},
	}).Err()
}

// PullTasks pulls tasks assigned to this worker from Redis Stream
func (b *TaskBus) PullTasks(ctx context.Context, workerID string, count int64) ([]*types.Task, error) {
	streams, err := b.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    b.consumerGroup,
		Consumer: workerID,
		Streams:  []string{b.streamKey, ">"},
		Count:    count,
		Block:    0,
	}).Result()

	if err != nil {
		return nil, err
	}

	var tasks []*types.Task
	for _, stream := range streams {
		for _, msg := range stream.Messages {
			// Check if this task is for this worker
			targetWorker, ok := msg.Values["worker"].(string)
			if !ok || targetWorker != workerID {
				continue
			}

			var task types.Task
			if err := json.Unmarshal([]byte(msg.Values["task"].(string)), &task); err != nil {
				continue
			}
			tasks = append(tasks, &task)

			// Acknowledge the message
			b.client.XAck(ctx, b.streamKey, b.consumerGroup, msg.ID)
		}
	}

	return tasks, nil
}

func (b *TaskBus) Close() error {
	return b.client.Close()
}
