package redis

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/kingxl111/mosprom/ApiGateway/internal/config"
)

type Client interface {
	Publish(ctx context.Context, channel string, message interface{}) error
	Subscribe(ctx context.Context, channels ...string) PubSub
	Close() error
}

type PubSub interface {
	Channel() <-chan *redis.Message
	Close() error
}

type client struct {
	rdb *redis.Client
}

type pubsub struct {
	ps *redis.PubSub
}

func NewClient(cfg config.RedisConfig) (Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password(),
		DB:       cfg.DB(),
	})

	// Проверяем соединение
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &client{rdb: rdb}, nil
}

func (c *client) Publish(ctx context.Context, channel string, message interface{}) error {
	return c.rdb.Publish(ctx, channel, message).Err()
}

func (c *client) Subscribe(ctx context.Context, channels ...string) PubSub {
	return &pubsub{ps: c.rdb.Subscribe(ctx, channels...)}
}

func (p *pubsub) Channel() <-chan *redis.Message {
	return p.ps.Channel()
}

func (p *pubsub) Close() error {
	return p.ps.Close()
}

func (c *client) Close() error {
	return c.rdb.Close()
}
