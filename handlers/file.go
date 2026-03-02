package handlers

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
)

// ServeFile serves a task image file. URL format: /here/file/{taskID}@{filename}
func (h *Handler) ServeFile(c fiber.Ctx) error {
	id := c.Params("id")
	parts := strings.SplitN(id, "@", 2)
	if len(parts) < 2 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}

	if _, err := strconv.Atoi(parts[0]); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "task ID must be a number"})
	}

	filePath := h.Cfg.MainPath + parts[0] + "/" + parts[1] + ".jpg"
	return c.SendFile(filePath)
}
