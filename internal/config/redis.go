package config

import (
	"os"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient() *redis.Client {
	redisAddr := os.Getenv("REDIS_ADDR")//env saves addr

	return redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
}