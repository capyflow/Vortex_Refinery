package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoStore struct {
	client   *mongo.Client
	database *mongo.Database
}

func NewMongoStore(uri, db string) (*MongoStore, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	return &MongoStore{
		client:   client,
		database: client.Database(db),
	}, nil
}

func (s *MongoStore) Close(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}

func (s *MongoStore) Collection(name string) *mongo.Collection {
	return s.database.Collection(name)
}
