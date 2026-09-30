package store

import (
	"context"
	"linkforge-url-shortener/internal/config"
	"time"

	"github.com/redis/go-redis/v9"
)

var redisClient = config.NewRedisClient()

func SaveSlug(slug string, url string, expiry time.Duration) (bool, error) {
	ctx := context.Background()

	saved, err := redisClient.SetNX(
		ctx,
		slug,
		url,
		expiry,
	).Result()

	return saved, err
}

func GetURL(slug string) (string, error) {
	ctx := context.Background()

	url, err := redisClient.Get(ctx, slug).Result()

	if err == redis.Nil {
		return "", redis.Nil
	}
	if err != nil {
		return "", err
	}
	return url, nil
}
func CacheURL(slug string, url string) error {
	return redisClient.Set(
		context.Background(),
		slug,
		url,
		24*time.Hour,
	).Err()
}
func GetRedisClient() *redis.Client {
	return redisClient
}
func CacheURLWithExpiry(slug string, url string, expiry time.Duration) error {
	return redisClient.Set(
		context.Background(),
		slug,
		url,
		expiry,
	).Err()
}
