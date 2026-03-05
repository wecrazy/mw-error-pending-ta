<div align="center">

# 🛠️ mw-pending-error-ta

**Middleware for Technical Assistance Error & Pending Data**

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![Fiber](https://img.shields.io/badge/Fiber-v3-00ACD7?style=flat-square&logo=go&logoColor=white)](https://gofiber.io)
[![MySQL](https://img.shields.io/badge/MySQL-8.0-4479A1?style=flat-square&logo=mysql&logoColor=white)](https://www.mysql.com)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](LICENSE)

Manages error & pending task data from technician field visits.  
Integrates with **Odoo ERP** and an internal **file store** service.

</div>

---

## 📦 Project Structure

```
mw-pending-error-ta/
│
├── main.go                    → Entry point & Fiber v3 route registration
│
├── config/
│   └── config.go              → Environment variable loader
│
├── database/
│   └── database.go            → MySQL connection setup
│
├── models/
│   └── models.go              → Request/response types, Odoo types, ImageMap
│
├── services/
│   ├── odoo.go                → Odoo API client (login, RPC calls, stage)
│   ├── filestore.go           → File store, folder ops, RemoveFalsyValues
│   └── task.go                → Task DB ops, logging, temp submissions
│
├── handlers/
│   ├── handler.go             → Handler struct (dependency injection)
│   ├── table.go               → List pending & error tasks
│   ├── submit.go              → Submit & edit task data
│   ├── data.go                → Get, check & delete task data
│   ├── reason.go              → Reason code listing & sync
│   ├── reload.go              → Reload/cleanup pending & error
│   ├── insert.go              → Insert tasks from external service
│   └── file.go                → Serve task image files
│
├── .env                       → Environment config (not committed)
├── go.mod / go.sum            → Go module files
└── LICENSE
```

---

## ⚙️ Requirements

| Dependency | Version |
|:-----------|:--------|
| Go         | 1.26+   |
| MySQL      | 5.7+    |
| Odoo ERP   | —       |
| File Store | —       |

---

## 🚀 Getting Started

**1.** Clone the repository

```bash
git clone <repo-url>
cd mw-pending-error-ta
```

**2.** Create `.env` from example

```env
# Database
DB_USER=
DB_PASS=
DB_HOST=
DB_PORT=
DB_NAME=

# Application
DATA_PATH=/opt/data_pro/
SERVER_PORT=:22441

# Odoo
ODOO_LOGIN_URL=
ODOO_GET_URL=
ODOO_UPDATE_URL=
ODOO_EMAIL=
ODOO_PASSWORD=

# File Store
FILESTORE_URL=
FILESTORE_FILE_URL=

# Notification
WA_WEGIL_URL=
```

**3.** Run

```bash
go mod tidy
go run main.go
```

> Server starts on the port specified by `SERVER_PORT` (default `:22441`)

---

## 📡 API Endpoints

### Data Tables

| Method | Endpoint | Description |
|:------:|:---------|:------------|
| `GET` | `/here/tablePending` | List all pending tasks |
| `GET` | `/here/tableError` | List all error tasks |

### Task Operations

| Method | Endpoint | Description |
|:------:|:---------|:------------|
| `POST` | `/here/postData` | Submit task data to Odoo |
| `POST` | `/here/editData` | Edit task data *(multipart form)* |
| `POST` | `/here/getData` | Read specific fields from `data.json` |
| `POST` | `/here/checkData` | Check Odoo stage & clean up if Done |
| `POST` | `/here/deleteData` | Delete task and all associated data |

### External Service Ingestion

| Method | Endpoint | Description |
|:------:|:---------|:------------|
| `POST` | `/here/insertDataError` | Insert error task from external service |
| `POST` | `/here/insertDataPending` | Insert pending task from external service |

### Reason Codes

| Method | Endpoint | Description |
|:------:|:---------|:------------|
| `POST` | `/here/listReason` | List reason codes by company |
| `GET` | `/here/reloadReason` | Sync reason codes from Odoo |

### Reload / Sync

| Method | Endpoint | Description |
|:------:|:---------|:------------|
| `GET` | `/here/reloadPending` | Remove pending tasks already in file store |
| `GET` | `/here/reloadError` | Remove error tasks already in file store |

### File Serving

| Method | Endpoint | Description |
|:------:|:---------|:------------|
| `GET` | `/here/file/:id` | Serve a task image (`{taskID}@{filename}`) |

---

## 🏗️ Architecture

```
┌──────────┐     ┌──────────────┐     ┌───────────┐
│  Client   │────▶│  Fiber v3    │────▶│  Handlers │
└──────────┘     │  HTTP Server │     └─────┬─────┘
                 └──────────────┘           │
                        ┌───────────────────┼───────────────────┐
                        ▼                   ▼                   ▼
                 ┌─────────────┐    ┌──────────────┐    ┌──────────────┐
                 │  Odoo ERP   │    │    MySQL     │    │  File Store  │
                 │  (JSON-RPC) │    │   Database   │    │   Service    │
                 └─────────────┘    └──────────────┘    └──────────────┘
```

---

<div align="center">
<sub>Built with ❤️ using Go & Fiber v3</sub>
</div>
