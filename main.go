package main

import (
	"log"

	"middleware-pending-error-ta/config"
	"middleware-pending-error-ta/database"
	"middleware-pending-error-ta/handlers"
	"middleware-pending-error-ta/services"

	"github.com/gofiber/fiber/v3"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	// Connect to database
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	log.Println("Successfully connected to the database!")

	// Initialize services
	odoo := services.NewOdooClient(cfg)
	task := services.NewTaskService(db, cfg, odoo)

	// Initialize handlers
	h := handlers.New(db, cfg, odoo, task)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		BodyLimit: 20 * 1024 * 1024, // 20 MB
	})

	// Register routes
	api := app.Group("/here")

	// Table listings (GET)
	api.Get("/tableError", h.ListError)
	api.Get("/tablePending", h.ListPending)

	// Data operations (POST)
	api.Post("/postData", h.SubmitData)
	api.Post("/editData", h.EditData)
	api.Post("/getData", h.GetData)
	api.Post("/checkData", h.CheckData)
	api.Post("/deleteData", h.DeleteData)
	api.Post("/listReason", h.ListReason)

	// Insert from external service (POST)
	api.Post("/insertDataError", h.InsertErrorData)
	api.Post("/insertDataPending", h.InsertPendingData)

	// File serving (GET)
	api.Get("/file/:id", h.ServeFile)

	// Reload / sync operations (GET)
	api.Get("/reloadReason", h.ReloadReason)
	api.Get("/reloadPending", h.ReloadPending)
	api.Get("/reloadError", h.ReloadError)

	// Start server
	log.Printf("Server running on port %s", cfg.ServerPort)
	log.Fatal(app.Listen(cfg.ServerPort))
}
