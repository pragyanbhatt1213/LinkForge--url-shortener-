package handlers

import (
	"linkforge-url-shortener/internal/store"

	"github.com/gofiber/fiber/v2"
)

func ListLinks(c *fiber.Ctx) error {
	links, err := store.GetLinks()

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(map[string]string{
			"error": "Failed to fetch links",
		})
	}

	return c.JSON(links)
}
