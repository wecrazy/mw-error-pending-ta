package handlers

import (
	"middleware-pending-error-ta/models"

	"github.com/gofiber/fiber/v3"
)

// ListPending returns all rows from the pending table (DataTables format).
func (h *Handler) ListPending(c fiber.Ctx) error {
	rows, err := h.DB.Query("SELECT id_task, time_start, time_stop, tid, teknisi FROM pending")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "query failed"})
	}
	defer rows.Close()

	var results []models.QueryResult
	for rows.Next() {
		var r models.QueryResult
		if err := rows.Scan(&r.IDTask, &r.TimeStart, &r.TimeStop, &r.TID, &r.Teknisi); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "scan failed"})
		}
		results = append(results, r)
	}

	return c.JSON(models.DataTablesResponse{Data: results})
}

// ListError returns all rows from the error table (DataTables format).
func (h *Handler) ListError(c fiber.Ctx) error {
	rows, err := h.DB.Query("SELECT id_task, time_start, time_stop, tid, teknisi FROM error")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "query failed"})
	}
	defer rows.Close()

	var results []models.QueryResult
	for rows.Next() {
		var r models.QueryResult
		if err := rows.Scan(&r.IDTask, &r.TimeStart, &r.TimeStop, &r.TID, &r.Teknisi); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "scan failed"})
		}
		results = append(results, r)
	}

	return c.JSON(models.DataTablesResponse{Data: results})
}
