package handlers

import (
	"linkforge-url-shortener/internal/services"
	"linkforge-url-shortener/internal/store"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

func Redirect(c *fiber.Ctx) error {

	// 1. Get slug from the URL
	slug := c.Params("slug")

	// 2. First check Redis
	url, err := store.GetURL(slug)

	if err == nil {
		// Redis HIT
		services.RecordClick(slug)

		return c.Redirect(url, fiber.StatusFound)
	}

	// 3. Redis had a real error
	if err != redis.Nil {
		return c.Status(fiber.StatusInternalServerError).JSON(map[string]string{
			"error": "Internal server error",
		})
	}

	// 4. Redis MISS → check PostgreSQL
	url, err = store.GetLink(slug)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(map[string]string{
			"error": "Short URL not found",
		})
	}

	// 5. PostgreSQL found it → restore Redis cache
	err = store.CacheURL(slug, url)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(map[string]string{
			"error": "Internal server error",
		})
	}

	// 6. Record the click
	services.RecordClick(slug)

	// 7. Redirect user
	return c.Redirect(url, fiber.StatusFound)
}