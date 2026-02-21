package db

import (
	"context"
	"log"
	"sync"

	"github.com/redis/go-redis/v9"
)

var (
	Redis     *redis.Client
	redisOnce sync.Once
)

func GetRedisInstance() *redis.Client {
	return Redis
}

func InitRedis(addr string) error {
	var err error
	redisOnce.Do(func() {
		Redis = redis.NewClient(&redis.Options{
			Addr: addr,
		})

		err = Redis.Ping(context.Background()).Err()
		if err != nil {
			log.Fatal(err)
		}
	})
	return err
}
