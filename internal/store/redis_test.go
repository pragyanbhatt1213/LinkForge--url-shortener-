package store

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestSaveSlug(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	redisClient = redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	saved, err := SaveSlug("abc123", "https://google.com", 24*time.Hour)

	if err != nil {
		t.Fatal(err)
	}

	if !saved {
		t.Error("expected slug to be saved")
	}

	savedAgain, err := SaveSlug("abc123", "https://youtube.com", 24*time.Hour)

	if err != nil {
		t.Fatal(err)
	}

	if savedAgain {
		t.Error("expected second save to fail because slug already exists")
	}

	url, err := GetURL("abc123")

	if err != nil {
		t.Fatal(err)
	}

	if url != "https://google.com" {
		t.Errorf("expected original URL, got %s", url)
	}
}

func TestGetURL(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	redisClient = redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	err = redisClient.Set(
		context.Background(),
		"abc123",
		"https://google.com",
		0,
	).Err()
	if err != nil {
		t.Fatal(err)
	}
	url, err := GetURL("abc123")
	if err != nil {
		t.Fatal(err)
	}
	if url != "https://google.com" {
		t.Errorf("expected https://google.com, got %s", url)
	}
}

func TestGetURLMissing(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	redisClient = redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	_, err = GetURL("doesnotexist")

	if err != redis.Nil {
		t.Errorf("expected redis.Nil, got %v", err)
	}
}
func TestGetURLRedisError(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}

	redisClient = redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// Shut down Redis
	mr.Close()

	_, err = GetURL("abc123")

	if err == nil {
		t.Fatal("expected Redis error, got nil")
	}

	if err == redis.Nil {
		t.Fatal("expected actual Redis error, got redis.Nil")
	}
}
func TestSaveSlugWithExpiry(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	redisClient = redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	saved, err := SaveSlug(
		"expiry-test",
		"https://example.com",
		time.Hour,
	)

	if err != nil {
		t.Fatal(err)
	}

	if !saved {
		t.Fatal("expected slug to be saved")
	}

	ttl := mr.TTL("expiry-test")

	if ttl <= 0 || ttl > time.Hour {
		t.Errorf("unexpected TTL: %v", ttl)
	}
}

// TestCacheMissAndRepopulate mirrors the exact three steps the redirect handler
// performs when Redis does not have a slug:
//
//  1. GetURL returns redis.Nil  (cache miss)
//  2. CacheURL writes the entry  (repopulate after PostgreSQL lookup)
//  3. GetURL now succeeds        (cache hit on the next request)
func TestCacheMissAndRepopulate(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	redisClient = redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	slug := "cache-miss-slug"
	want := "https://example.com/cache-miss"

	// Step 1: key does not exist yet → must be a cache miss (redis.Nil)
	_, err = GetURL(slug)
	if err != redis.Nil {
		t.Fatalf("expected redis.Nil on miss, got %v", err)
	}

	// Step 2: simulate the redirect handler writing the URL back into Redis
	// after fetching it from PostgreSQL
	if err = CacheURL(slug, want); err != nil {
		t.Fatalf("CacheURL failed: %v", err)
	}

	// Step 3: the next lookup must now return the URL (cache hit)
	got, err := GetURL(slug)
	if err != nil {
		t.Fatalf("GetURL after repopulate failed: %v", err)
	}
	if got != want {
		t.Errorf("expected %q after repopulate, got %q", want, got)
	}
}

