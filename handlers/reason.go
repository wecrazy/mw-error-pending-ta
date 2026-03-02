package handlers

import (
	"database/sql"
	"encoding/json"
	"log"

	"middleware-pending-error-ta/models"

	"github.com/gofiber/fiber/v3"
)

// ListReason returns reason codes as a JSON object for a given company.
func (h *Handler) ListReason(c fiber.Ctx) error {
	var req models.RequestDataReason
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON"})
	}
	if req.Company == "" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "company is required"})
	}

	var jsonResult sql.NullString
	err := h.DB.QueryRow(
		"SELECT JSON_OBJECTAGG(reason_id, name) AS json_result FROM check_reason WHERE com = ?",
		req.Company,
	).Scan(&jsonResult)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "database error"})
	}
	if !jsonResult.Valid {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "company not found"})
	}

	c.Set("Content-Type", "application/json")
	return c.SendString(jsonResult.String)
}

// ReloadReason fetches fresh reason codes from Odoo and repopulates the check_reason table.
func (h *Handler) ReloadReason(c fiber.Ctx) error {
	body := `{"jsonrpc":"2.0","params":{"model":"x_reason_code","fields":["x_name","x_company_id","x_reason_code"],"domain":[["x_active","=",1]],"order":"create_date asc"}}`

	_, cookies, err := h.Odoo.Login("testmfjr@gmail.com", "Ma113060111!")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "odoo not responding"})
	}

	jsonData, err := h.Odoo.Call(h.Cfg.OdooGetURL, []byte(body), cookies)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "odoo call failed"})
	}

	var response models.OdooReasonResponse
	if err := json.Unmarshal([]byte(jsonData), &response); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to parse odoo response"})
	}

	tx, err := h.DB.Begin()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "database error"})
	}

	if _, err := tx.Exec("DELETE FROM check_reason"); err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "database error"})
	}

	stmt, err := tx.Prepare("INSERT INTO check_reason (reason_id, company_id, name, com) VALUES (?, ?, ?, ?)")
	if err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "database error"})
	}
	defer stmt.Close()

	for _, item := range response.Result {
		companyID, ok := item.XCompanyID[0].(float64)
		if !ok {
			continue
		}
		companyName, _ := item.XCompanyID[1].(string)
		if _, err := stmt.Exec(item.ID, int(companyID), item.XName, companyName); err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "database insert failed"})
		}
		log.Printf("Inserted reason: id=%d, company=%d, name=%s", item.ID, int(companyID), item.XName)
	}

	if err := tx.Commit(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "database commit failed"})
	}

	return c.SendString("Reason codes reloaded successfully")
}
