package pkgredis

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient interface {
	Connect() (*redis.Client, error)
}

type redisClient struct {
	redisConfig
	redisInstance *redis.Client
	mu            sync.Mutex
}

// Connect implements RedisClient.
func (r *redisClient) Connect() (*redis.Client, error) {
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{
		Addr:     r.hostAddress,
		Password: r.password,
		DB:       r.dbNumber,
	})
	r.redisInstance = client
	maxRetry := 10

	if r.retryConnect {
		i := 1
		for {
			if i > maxRetry {
				return nil, fmt.Errorf("cannot connect redis instance")
			}
			_, err := client.Ping(ctx).Result()
			if err == nil {
				break
			}
			log.Printf("redis err: %s connecting to %s retry %d", err, r.hostAddress, i)
			i += 1
			time.Sleep(2000 * time.Millisecond)
		}
	}

	return r.redisInstance, nil
}

func NewRedisClient(opts ...RedisOption) RedisClient {
	rdsCfg := &redisConfig{
		hostAddress:  "localhost",
		dbNumber:     0,
		retryConnect: false,
	}

	for _, opt := range opts {
		opt(rdsCfg)
	}

	return &redisClient{
		redisConfig:   *rdsCfg,
		redisInstance: &redis.Client{},
		mu:            sync.Mutex{},
	}
}
