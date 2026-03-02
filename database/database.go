package database

import (
"database/sql"
"fmt"

_ "github.com/go-sql-driver/mysql"
"middleware-pending-error-ta/config"
)

// Connect establishes a MySQL database connection and verifies it with a ping.
func Connect(cfg *config.Config) (*sql.DB, error) {
dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
cfg.DBUser, cfg.DBPass, cfg.DBHost, cfg.DBPort, cfg.DBName,
)

db, err := sql.Open("mysql", dsn)
if err != nil {
return nil, fmt.Errorf("failed to open database: %w", err)
}

if err := db.Ping(); err != nil {
return nil, fmt.Errorf("database ping failed: %w", err)
}

return db, nil
}
