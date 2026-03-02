package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"

	"middleware-pending-error-ta/models"
	"middleware-pending-error-ta/services"

	"github.com/gofiber/fiber/v3"
)

// SubmitData authenticates the user, posts task data to Odoo, and cleans up.
func (h *Handler) SubmitData(c fiber.Ctx) error {
	var req models.RequestData
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON"})
	}
	if req.IDTask == "" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "id_task is required"})
	}

	isLogin, cookies, err := h.Odoo.Login(req.Email, req.Password)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
	}
	if !isLogin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "failed to login to Odoo"})
	}

	stage, err := h.Odoo.GetStage(req.IDTask, cookies)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
	}
	if stage == "Cancel" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Status is " + stage})
	}

	// Read and parse data.json
	dataPath := h.Cfg.MainPath + req.IDTask + "/data.json"
	data, err := os.ReadFile(dataPath)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "data.json not found"})
	}

	var jsonObject map[string]interface{}
	if err := json.Unmarshal(data, &jsonObject); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "failed to parse data.json"})
	}

	params, ok := jsonObject["params"].(map[string]interface{})
	if !ok {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "invalid data.json structure"})
	}

	if req.IsPaid {
		params["x_paid"] = true
	}

	data, err = json.Marshal(jsonObject)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "failed to marshal data"})
	}

	// Post to file store
	status, fsErr := services.PostToFileStore(h.Cfg.FileStoreURL, string(data))
	if fsErr != nil || status != 200 {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "failed to post to file store"})
	}

	// Handle keep data / temp submission
	method := "Submit"
	if req.KeepData {
		method = "Submit (Temp)"
		h.Task.UpsertTempSubmission(req.Email, method, req.IDTask, "")
	}
	h.Task.InsertLog(req.Email, method, req.IDTask, "", "")

	// Clean data and post to Odoo (if not already Done)
	var dataMap map[string]interface{}
	if err := json.Unmarshal(data, &dataMap); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "failed to parse data"})
	}
	services.RemoveFalsyValues(dataMap)
	cleanData, _ := json.Marshal(dataMap)

	if stage != "Done" {
		if err := h.Odoo.PostUpdate(cleanData, cookies); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "failed to post to Odoo: " + err.Error()})
		}
	}

	// Clean up database records
	if err := h.Task.DeleteFromTables(req.IDTask, "error", "pending"); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "database error"})
	}

	if !req.KeepData {
		services.DeleteFolder(h.Cfg.MainPath + req.IDTask)
	}

	return c.JSON(fiber.Map{"message": "Data received successfully"})
}

// EditData handles multipart form edits to a task's data.json and photos.
func (h *Handler) EditData(c fiber.Ctx) error {
	// Parse form values
	name := c.FormValue("name")
	password := c.FormValue("password")
	task := c.FormValue("task")
	paid := c.FormValue("isPaid")
	thermal := c.FormValue("thermal")

	if name == "" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "name is required"})
	}
	if password == "" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "password is required"})
	}
	if task == "" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "task is required"})
	}
	if _, err := strconv.Atoi(task); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "task must be a number"})
	}

	thermalInt := 0
	if thermal != "" {
		v, err := strconv.Atoi(thermal)
		if err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "thermal must be a number"})
		}
		thermalInt = v
	}

	logEdit := c.FormValue("logEdit")
	if logEdit != "" && name != "" {
		logEdit += " ~" + name
	}

	keepData := c.FormValue("keepData") == "true"
	isFinal := c.FormValue("is_final") == "true"

	// Authenticate with Odoo
	isLogin, cookies, err := h.Odoo.Login(name, password)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
	}
	if !isLogin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "failed to login to Odoo"})
	}

	// Verify folder and data.json exist
	taskDir := h.Cfg.MainPath + "/" + task
	dataPath := taskDir + "/data.json"
	if _, err := os.Stat(taskDir); os.IsNotExist(err) {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "task folder not found"})
	}
	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "data.json not found"})
	}

	// Read and parse data.json
	fileData, err := os.ReadFile(dataPath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to read data.json"})
	}

	var jsonObject map[string]interface{}
	if err := json.Unmarshal(fileData, &jsonObject); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "invalid data.json"})
	}

	params, ok := jsonObject["params"].(map[string]interface{})
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "invalid data.json structure"})
	}

	// Update paid status
	if paid == "true" {
		params["x_paid"] = true
	} else {
		delete(params, "x_paid")
	}

	// Update thermal supply
	if thermal != "" {
		params["x_supply_thermal"] = thermalInt
	}

	// Handle file uploads
	for _, field := range models.ImageFields {
		fileHeader, err := c.FormFile(field)
		if err != nil {
			continue // No file uploaded for this field
		}

		f, err := fileHeader.Open()
		if err != nil {
			continue
		}

		imageData, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			continue
		}

		// Update params with base64
		base64Str := base64.StdEncoding.EncodeToString(imageData)
		params[field] = base64Str

		// Save file to disk
		dst := fmt.Sprintf("%s/%s/%s.jpg", h.Cfg.MainPath, task, field)
		if err := os.WriteFile(dst, imageData, 0644); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save file"})
		}
	}

	// Update reason and keterangan
	reason := c.FormValue("reason")
	keterangan := c.FormValue("keterangan")

	if reason != "" {
		numReason, err := strconv.Atoi(reason)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "reason must be a number"})
		}
		params["x_reason_code_id"] = float64(numReason)
	}

	if keterangan != "" {
		params["x_keterangan"] = keterangan
	}

	// Write updated data.json
	updatedJSON, err := json.MarshalIndent(jsonObject, "", "    ")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to marshal JSON"})
	}
	if err := os.WriteFile(dataPath, updatedJSON, 0644); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to write data.json"})
	}

	// Get stage from Odoo
	stage, err := h.Odoo.GetStage(task, cookies)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
	}
	if stage == "Cancel" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Status is " + stage})
	}

	// Re-read updated file and post to file store
	dataB, err := os.ReadFile(h.Cfg.MainPath + task + "/data.json")
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "failed to read data.json"})
	}

	status, fsErr := services.PostToFileStore(h.Cfg.FileStoreURL, string(dataB))
	if fsErr != nil || status != 200 {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "failed to post to file store"})
	}

	// Determine method and handle temp submission
	method := "edit"
	if keepData && isFinal {
		method = "edit (Final)"
	} else if keepData {
		method = "edit (Temp)"
		h.Task.UpsertTempSubmission(name, method, task, logEdit)
	}
	h.Task.InsertLog(name, method, task, "", logEdit)

	// Remove falsy values and post to Odoo
	var dataMap map[string]interface{}
	json.Unmarshal(dataB, &dataMap)
	services.RemoveFalsyValues(dataMap)
	cleanData, _ := json.Marshal(dataMap)

	if isFinal {
		if err := h.Odoo.PostUpdate(cleanData, cookies); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "failed to post to Odoo: " + err.Error()})
		}
		h.Task.DeleteFromTables(task, "temp_submission")
	} else if stage != "Done" {
		if err := h.Odoo.PostUpdate(cleanData, cookies); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "failed to post to Odoo: " + err.Error()})
		}
	}

	// Clean up database records
	if err := h.Task.DeleteFromTables(task, "error", "pending"); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "database error"})
	}

	if !keepData {
		services.DeleteFolder(h.Cfg.MainPath + task)
	}

	return c.JSON(fiber.Map{"message": "Data received successfully"})
}
