package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"middleware-pending-error-ta/models"

	"github.com/gofiber/fiber/v3"
)

// insertTaskData is a shared handler for inserting error and pending task data.
// It decodes base64 images, saves them to disk, stores data.json,
// resolves reason names, and upserts into the target table.
func (h *Handler) insertTaskData(c fiber.Ctx, tableName string) error {
	// Define required headers
	requiredHeaders := []string{
		"tid", "tech", "com", "res", "wo", "tip", "mer",
		"ket", "tik", "tit", "sla", "spk", "rcv",
		"tid2", "mid", "edc", "sn", "alamat",
	}
	if tableName == "error" {
		// Error table also requires "als" (problem/description)
		requiredHeaders = []string{
			"tid", "tech", "com", "res", "wo", "tip", "mer", "als",
			"ket", "tik", "tit", "sla", "spk", "rcv",
			"tid2", "mid", "edc", "sn", "alamat",
		}
	}

	// Collect and validate headers
	headers := make(map[string]string)
	for _, hdr := range requiredHeaders {
		val := c.Get(hdr)
		if val == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing header: " + hdr})
		}
		headers[hdr] = val
	}

	// Parse body
	body := c.Body()
	var bodyObject models.TaskData
	if err := json.Unmarshal(body, &bodyObject); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	task := bodyObject.Params.ID

	// Check if task already exists in file store
	checkURL := h.Cfg.FileStoreURL1 + "/" + task + "@x_foto_edc"
	resp, err := http.Get(checkURL)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file store check failed"})
	}
	resp.Body.Close()

	if resp.StatusCode == 200 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "already exists"})
	}

	// Create task directory
	taskDir := h.Cfg.MainPath + task
	if err := os.MkdirAll(taskDir, os.ModePerm); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create task directory"})
	}

	// Save images from base64
	imageMap := bodyObject.Params.ImageMap()
	for _, field := range models.ImageFields {
		data := imageMap[field]
		if len(data) < 2 {
			continue
		}

		decoded, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "invalid base64 for " + field})
		}

		if err := os.WriteFile(fmt.Sprintf("%s/%s.jpg", taskDir, field), decoded, 0644); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save image"})
		}
	}

	// Save data.json
	if err := os.WriteFile(fmt.Sprintf("%s/data.json", taskDir), body, 0644); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save data.json"})
	}

	// Resolve reason name
	com, err := strconv.Atoi(headers["com"])
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "com must be a number"})
	}
	res, err := strconv.Atoi(headers["res"])
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "res must be a number"})
	}

	reasonName, companyName, err := h.Task.GetReasonName(com, res)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "reason not found"})
	}

	// Delete from opposite table
	oppositeTable := "pending"
	if tableName == "pending" {
		oppositeTable = "error"
	}
	h.Task.CheckAndDeleteExisting(task, oppositeTable)

	// Upsert into target table
	var count int
	h.DB.QueryRow("SELECT COUNT(*) FROM "+tableName+" WHERE id_task = ?", task).Scan(&count)

	var dbErr error
	if count > 0 {
		dbErr = h.updateTaskRecord(tableName, task, headers, bodyObject, reasonName, companyName)
	} else {
		dbErr = h.insertTaskRecord(tableName, task, headers, bodyObject, reasonName, companyName)
	}

	if dbErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": dbErr.Error()})
	}

	return c.SendString("Data inserted successfully")
}

// updateTaskRecord updates an existing record in the error or pending table.
func (h *Handler) updateTaskRecord(tableName, task string, headers map[string]string, body models.TaskData, reasonName, companyName string) error {
	if tableName == "error" {
		_, err := h.DB.Exec(`UPDATE error SET
			time_start=?, time_stop=?, tid=?, teknisi=?, reason=?, company=?,
			wo=?, merchant=?, type=?, keterangan=?, `+"`desc`"+`=?, sla=?, type2=?,
			problem=?, spk=?, receiveDate=?, mid=?, alamat=?, edc_type=?, sn=?, tid_bank=?
			WHERE id_task=?`,
			body.Params.TimesheetTimerFirstStart, body.Params.TimesheetTimerLastStop,
			headers["tid"], headers["tech"], reasonName, companyName,
			headers["wo"], headers["mer"], headers["tip"], headers["ket"],
			headers["tit"], headers["sla"], headers["tik"],
			headers["als"], headers["spk"], headers["rcv"],
			headers["mid"], headers["alamat"], headers["edc"], headers["sn"], headers["tid2"],
			task,
		)
		return err
	}

	_, err := h.DB.Exec(`UPDATE pending SET
		time_start=?, time_stop=?, tid=?, teknisi=?, reason=?, company=?,
		wo=?, merchant=?, type=?, keterangan=?, `+"`desc`"+`=?, sla=?, type2=?,
		spk=?, receiveDate=?, mid=?, alamat=?, edc_type=?, sn=?, tid_bank=?
		WHERE id_task=?`,
		body.Params.TimesheetTimerFirstStart, body.Params.TimesheetTimerLastStop,
		headers["tid"], headers["tech"], reasonName, companyName,
		headers["wo"], headers["mer"], headers["tip"], headers["ket"],
		headers["tit"], headers["sla"], headers["tik"],
		headers["spk"], headers["rcv"],
		headers["mid"], headers["alamat"], headers["edc"], headers["sn"], headers["tid2"],
		task,
	)
	return err
}

// insertTaskRecord inserts a new record into the error or pending table.
func (h *Handler) insertTaskRecord(tableName, task string, headers map[string]string, body models.TaskData, reasonName, companyName string) error {
	if tableName == "error" {
		_, err := h.DB.Exec(`INSERT INTO error
			(id_task, time_start, time_stop, tid, teknisi, reason, company, wo, merchant,
			 type, keterangan, `+"`desc`"+`, sla, type2, problem, spk, receiveDate,
			 mid, alamat, edc_type, sn, tid_bank)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			task, body.Params.TimesheetTimerFirstStart, body.Params.TimesheetTimerLastStop,
			headers["tid"], headers["tech"], reasonName, companyName,
			headers["wo"], headers["mer"], headers["tip"], headers["ket"],
			headers["tit"], headers["sla"], headers["tik"],
			headers["als"], headers["spk"], headers["rcv"],
			headers["mid"], headers["alamat"], headers["edc"], headers["sn"], headers["tid2"],
		)
		return err
	}

	_, err := h.DB.Exec(`INSERT INTO pending
		(id_task, time_start, time_stop, tid, teknisi, reason, company, wo, merchant,
		 type, keterangan, `+"`desc`"+`, sla, type2, spk, receiveDate,
		 mid, alamat, edc_type, sn, tid_bank)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		task, body.Params.TimesheetTimerFirstStart, body.Params.TimesheetTimerLastStop,
		headers["tid"], headers["tech"], reasonName, companyName,
		headers["wo"], headers["mer"], headers["tip"], headers["ket"],
		headers["tit"], headers["sla"], headers["tik"],
		headers["spk"], headers["rcv"],
		headers["mid"], headers["alamat"], headers["edc"], headers["sn"], headers["tid2"],
	)
	return err
}

// InsertErrorData handles insertion of error task data (with problem field).
func (h *Handler) InsertErrorData(c fiber.Ctx) error {
	return h.insertTaskData(c, "error")
}

// InsertPendingData handles insertion of pending task data.
func (h *Handler) InsertPendingData(c fiber.Ctx) error {
	return h.insertTaskData(c, "pending")
}
