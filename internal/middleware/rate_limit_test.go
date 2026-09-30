package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestCheckRateLimitUnderLimit(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	key := "rate_limit:test"

	for i := 1; i <= 10; i++ {
		count, err := CheckRateLimit(client, key)

		if err != nil {
			t.Fatal(err)
		}

		if count != int64(i) {
			t.Errorf("expected count %d, got %d", i, count)
		}
	}
}

func TestCheckRateLimitOverLimit(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	key := "rate_limit:test"

	for i := 0; i < 10; i++ {
		_, err := CheckRateLimit(client, key)

		if err != nil {
			t.Fatal(err)
		}
	}

	count, err := CheckRateLimit(client, key)

	if err != nil {
		t.Fatal(err)
	}

	if count != 11 {
		t.Errorf("expected count 11, got %d", count)
	}
}

func TestCheckRateLimitResetsAfterTTL(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	key := "rate_limit:test"

	count, err := CheckRateLimit(client, key)

	if err != nil {
		t.Fatal(err)
	}

	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	// Make sure the key has an expiration.
	ttl, err := client.TTL(
		context.Background(),
		key).Result()

	if err != nil {
		t.Fatal(err)
	}

	if ttl <= 0 {
		t.Error("expected rate limit key to have an expiration")
	}

	// Move miniredis time forward by one minute.
	mr.FastForward(time.Minute)

	count, err = CheckRateLimit(client, key)

	if err != nil {
		t.Fatal(err)
	}

	if count != 1 {
		t.Errorf("expected counter to reset to 1, got %d", count)
	}
}
