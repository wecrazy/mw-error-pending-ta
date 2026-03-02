package handlers

import (
	"database/sql"

	"middleware-pending-error-ta/config"
	"middleware-pending-error-ta/services"
)

// Handler holds shared dependencies for all HTTP handlers.
type Handler struct {
	DB   *sql.DB
	Cfg  *config.Config
	Odoo *services.OdooClient
	Task *services.TaskService
}

// New creates a new Handler with all dependencies.
func New(db *sql.DB, cfg *config.Config, odoo *services.OdooClient, task *services.TaskService) *Handler {
	return &Handler{
		DB:   db,
		Cfg:  cfg,
		Odoo: odoo,
		Task: task,
	}
}
