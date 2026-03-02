package handlers

import (
	"github.com/gofiber/fiber/v3"
)

// ReloadPending checks pending tasks against the file store and removes completed ones.
func (h *Handler) ReloadPending(c fiber.Ctx) error {
	if err := h.Task.ReloadTasks("pending"); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendString("good")
}

// ReloadError checks error tasks against the file store and removes completed ones.
func (h *Handler) ReloadError(c fiber.Ctx) error {
	if err := h.Task.ReloadTasks("error"); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendString("good")
}
