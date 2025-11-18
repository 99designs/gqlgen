package dbconnection

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/cache/v9"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type Cache struct {
	Cache *cache.Cache
}

const DefaultRedisDB = 0

func NewCacheConnection() (*Cache, error) {
	const op = "cmd.dbconnection.NewCacheConnection()"

	redisDB, err := strconv.Atoi(os.Getenv("REDIS_DB"))
	if err != nil {
		log.Warn().Msg("Failed get redis db from env")
		redisDB = DefaultRedisDB
	}
	ring := redis.NewRing(&redis.RingOptions{
		Addrs: map[string]string{
			"shard1": os.Getenv("REDIS_HOST1") + ":" + os.Getenv("REDIS_PORT1"),
			"shard2": os.Getenv("REDIS_HOST2") + ":" + os.Getenv("REDIS_PORT2"),
		},

		DB: redisDB,

		Password: os.Getenv("REDIS_PASSWORD"),

		MaxRetries: 3,

		DialTimeout: 50 * time.Millisecond,
	})

	if err := ring.Ping(context.TODO()).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %v", err)
	}

	cache := &Cache{
		Cache: cache.New(&cache.Options{
			Redis:      ring,
			LocalCache: cache.NewTinyLFU(1000, time.Minute),
		}),
	}

	return cache, nil
}
