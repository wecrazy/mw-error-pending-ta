package handlers

import (
	"encoding/json"
	"os"
	"strconv"

	"middleware-pending-error-ta/models"
	"middleware-pending-error-ta/services"

	"github.com/gofiber/fiber/v3"
)

// GetData reads specific fields from a task's data.json file.
func (h *Handler) GetData(c fiber.Ctx) error {
	var req models.RequestDataJSON
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON"})
	}

	task := req.IDTask
	if task == "" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "id_task is required"})
	}
	if _, err := strconv.Atoi(task); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "task must be a number"})
	}

	dataPath := h.Cfg.MainPath + "/" + task + "/data.json"
	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "data.json not found"})
	}

	raw, err := os.ReadFile(dataPath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to read data.json"})
	}

	var fileData models.JSONFile
	if err := json.Unmarshal(raw, &fileData); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "invalid data.json"})
	}

	var result []models.DataFieldValue
	for _, d := range req.Data {
		val, ok := fileData.Params[d.Name]
		if !ok {
			// Set default based on expected type
			switch d.Type {
			case "string":
				val = ""
			case "integer":
				val = 0
			case "boolean":
				val = false
			default:
				val = nil
			}
		} else if d.Type == "array" {
			if arr, ok2 := val.([]interface{}); ok2 && len(arr) > d.Index {
				val = arr[d.Index]
			} else {
				val = nil
			}
		}

		result = append(result, models.DataFieldValue{Name: d.Name, Data: val})
	}

	return c.JSON(models.ResponseDataJSON{IDTask: req.IDTask, Result: result})
}

// CheckData verifies a task's Odoo stage and cleans up if "Done".
func (h *Handler) CheckData(c fiber.Ctx) error {
	var req models.RequestData
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON"})
	}
	if req.IDTask == "" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "id_task is required"})
	}

	// Use service account for check
	_, cookies, err := h.Odoo.Login("testmfjr@gmail.com", "Ma113060111!")
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "odoo not responding"})
	}

	stage, err := h.Odoo.GetStage(req.IDTask, cookies)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
	}

	if stage != "Done" {
		return c.JSON(fiber.Map{"message": "Status is " + stage})
	}

	// Stage is "Done" — clean up all tables and folder
	h.Task.DeleteFromTables(req.IDTask, "error", "pending", "temp_submission")
	services.DeleteFolder(h.Cfg.MainPath + req.IDTask)

	return c.JSON(fiber.Map{"message": "Status is " + stage})
}

// DeleteData authenticates the user, logs the deletion, and removes all task data.
func (h *Handler) DeleteData(c fiber.Ctx) error {
	var req models.RequestData
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON"})
	}

	isLogin, _, err := h.Odoo.Login(req.Email, req.Password)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
	}
	if !isLogin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "failed to login to Odoo"})
	}

	if err := h.Task.InsertLog(req.Email, "Delete", req.IDTask, req.Reason, ""); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "failed to insert log"})
	}

	h.Task.DeleteFromTables(req.IDTask, "error", "pending", "temp_submission")
	services.DeleteFolder(h.Cfg.MainPath + req.IDTask)

	return c.JSON("DONE")
}
