package handlers

import (
	"os"

	"github.com/gofiber/fiber/v2"

	"linkforge-url-shortener/internal/models"
	"linkforge-url-shortener/internal/services"
)

func Shorten(c *fiber.Ctx) error {
	var req models.ShortenRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]string{
			"error": "Invalid request body",
		})
	}

	slug, err := services.CreateShortURL(req.URL, req.CustomSlug, req.ExpiryHours)
	if err != nil {
		if err.Error() == "slug already exists" {
			return c.Status(fiber.StatusConflict).JSON(map[string]string{
				"error": "Custom slug already exists",
			})
		}

		if err.Error() == "expiry must be greater than 0" ||
			err.Error() == "custom slug must be between 3 and 50 characters" {
			return c.Status(fiber.StatusBadRequest).JSON(map[string]string{
				"error": err.Error(),
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(map[string]string{
			"error": "Failed to create short URL",
		})
	}
	baseURL := os.Getenv("BASE_URL")
	return c.JSON(map[string]string{
		"short_url": baseURL + "/" + slug,
	})
}
