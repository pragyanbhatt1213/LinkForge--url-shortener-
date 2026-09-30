package middleware

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

func CheckRateLimit(redisClient *redis.Client, key string) (int64, error) {
	count, err := redisClient.Incr(
		context.Background(),
		key,
	).Result()

	if err != nil {
		return 0, err
	}

	if count == 1 {
		err := redisClient.Expire(
			context.Background(),
			key,
			time.Minute,
		).Err()

		if err != nil {
			return 0, err
		}
	}

	return count, nil
}

func RateLimit(redisClient *redis.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ip := c.IP()

		key := "rate_limit:" + ip

		count, err := CheckRateLimit(redisClient, key)

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(
				map[string]string{
					"error": "Rate limiter unavailable",
				},
			)
		}

		if count > 10 {
			return c.Status(fiber.StatusTooManyRequests).JSON(
				map[string]string{
					"error": "Too many requests",
				},
			)
		}

		c.Set("X-RateLimit-Limit", "10")
		c.Set(
			"X-RateLimit-Remaining",
			strconv.FormatInt(10-count, 10),
		)

		return c.Next()
	}
}
