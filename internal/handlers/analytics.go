package handlers

import (
	"linkforge-url-shortener/internal/store"

	"github.com/gofiber/fiber/v2"
)

func Analytics(c *fiber.Ctx) error {
	slug := c.Params("slug")

	count, err := store.GetClickCount(slug)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(map[string]string{
			"error": "Failed to get analytics",
		})
	}

	return c.JSON(map[string]interface{}{
		"slug":   slug,
		"clicks": count,
	})
}
