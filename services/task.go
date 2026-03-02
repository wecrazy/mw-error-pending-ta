package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"middleware-pending-error-ta/config"
	"middleware-pending-error-ta/models"
)

// TaskService handles database operations for tasks, logs, and temp submissions.
type TaskService struct {
	DB   *sql.DB
	Cfg  *config.Config
	Odoo *OdooClient
}

// NewTaskService creates a new TaskService.
func NewTaskService(db *sql.DB, cfg *config.Config, odoo *OdooClient) *TaskService {
	return &TaskService{DB: db, Cfg: cfg, Odoo: odoo}
}

// GetReasonName returns the reason name and company name for a given company/reason ID pair.
func (s *TaskService) GetReasonName(companyID, reasonID int) (string, string, error) {
	var name, com string
	err := s.DB.QueryRow(
		"SELECT name, com FROM check_reason WHERE company_id = ? AND reason_id = ? LIMIT 1",
		companyID, reasonID,
	).Scan(&name, &com)

	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", fmt.Errorf("no record for company_id=%d, reason_id=%d", companyID, reasonID)
		}
		return "", "", err
	}
	return name, com, nil
}

// DeleteFromTables removes a task from the specified database tables.
func (s *TaskService) DeleteFromTables(idTask string, tables ...string) error {
	for _, table := range tables {
		if _, err := s.DB.Exec("DELETE FROM "+table+" WHERE id_task = ?", idTask); err != nil {
			return fmt.Errorf("failed to delete from %s: %w", table, err)
		}
	}
	return nil
}

// CheckAndDeleteExisting removes a task from the given table if it exists (silent on error).
func (s *TaskService) CheckAndDeleteExisting(idTask, table string) {
	if idTask == "" || table == "" {
		return
	}
	s.DB.Exec("DELETE FROM "+table+" WHERE id_task = ?", idTask)
}

// ReloadTasks checks all tasks in the given table and removes those already
// present in the file store (i.e. successfully submitted).
func (s *TaskService) ReloadTasks(tableName string) error {
	rows, err := s.DB.Query("SELECT id_task FROM " + tableName)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var idTask string
		if err := rows.Scan(&idTask); err != nil {
			log.Printf("Error scanning row from %s: %v", tableName, err)
			continue
		}

		checkURL := s.Cfg.FileStoreURL1 + "/" + idTask + "@x_foto_edc"
		resp, err := http.Get(checkURL)
		if err != nil {
			log.Printf("HTTP request failed for task %s: %v", idTask, err)
			continue
		}
		io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == 200 {
			if err := s.DeleteFromTables(idTask, "error", "pending"); err != nil {
				return err
			}
			if err := DeleteFolder(s.Cfg.MainPath + idTask); err != nil {
				return err
			}
		}
	}

	return rows.Err()
}

// InsertLog records an action log and sends a WA notification.
func (s *TaskService) InsertLog(email, method, id, reason, logEdit string) error {
	data, typeCase, err := s.FindTaskData(id)
	if err != nil {
		return err
	}

	// Parse date_on_check
	var dateToInsert sql.NullTime
	if data.DateCheck != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", data.DateCheck); err == nil {
			dateToInsert = sql.NullTime{Time: t, Valid: true}
		}
	}

	// Send WA notification (fire and forget)
	now := time.Now().Format("2006-01-02 15:04:05")
	waPayload, _ := json.Marshal(map[string]string{
		"email":      email,
		"method":     method,
		"wo":         data.WO,
		"spk":        data.SPK,
		"technician": data.Teknisi,
		"type_case":  typeCase,
		"problem":    data.Problem,
		"mid":        data.Mid,
		"tid":        data.TID,
		"rc":         data.Reason,
		"reason":     reason,
		"date":       now,
	})
	s.Odoo.Call(s.Cfg.WegilURL, waPayload, nil)

	// Build dynamic insert query
	columns := []string{
		"email", "method", "wo", "spk", "teknisi", "type_case", "problem",
		"type", "type2", "sla", "rc", "tid", "keterangan", "reason",
		"mid", "alamat", "edc_type", "sn", "tid_bank",
		"date_on_check", "date_in_dashboard", "ta_feedback",
	}
	args := []interface{}{
		email, method, data.WO, data.SPK, data.Teknisi, typeCase, data.Problem,
		data.Type, data.Type2, data.SLA, data.Reason, data.TID, data.Keterangan, reason,
		data.Mid, data.Alamat, data.EdcType, data.Sn, data.TID2,
		dateToInsert, data.Date, data.TaFB,
	}

	if logEdit != "" && (strings.EqualFold(method, "edit") || strings.EqualFold(method, "edit (temp)")) {
		columns = append(columns, "log_edit")
		args = append(args, logEdit)
	}

	placeholders := make([]string, len(columns))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	query := "INSERT INTO log_act (" + strings.Join(columns, ",") + ") VALUES (" + strings.Join(placeholders, ",") + ")"
	if _, err := s.DB.Exec(query, args...); err != nil {
		return err
	}

	return nil
}

// FindTaskData searches for task data across error, pending, and temp_submission tables.
func (s *TaskService) FindTaskData(idTask string) (models.ErrorData, string, error) {
	var data models.ErrorData
	var dateOnCheck, dateInDashboard sql.NullString

	// Try error table (includes problem field)
	err := s.DB.QueryRow(`
		SELECT IFNULL(wo,''), IFNULL(spk,''), IFNULL(teknisi,''), IFNULL(problem,''),
			IFNULL(type,''), IFNULL(type2,''), IFNULL(sla,''), IFNULL(reason,''),
			IFNULL(tid,''), IFNULL(keterangan,''), IFNULL(mid,''), IFNULL(alamat,''),
			IFNULL(edc_type,''), IFNULL(sn,''), IFNULL(tid_bank,''),
			IFNULL(date_on_check,''), IFNULL(date,''), IFNULL(ta_feedback,'')
		FROM error WHERE id_task = ?`, idTask,
	).Scan(
		&data.WO, &data.SPK, &data.Teknisi, &data.Problem,
		&data.Type, &data.Type2, &data.SLA, &data.Reason,
		&data.TID, &data.Keterangan, &data.Mid, &data.Alamat,
		&data.EdcType, &data.Sn, &data.TID2,
		&dateOnCheck, &dateInDashboard, &data.TaFB,
	)
	if err == nil {
		data.DateCheck = dateOnCheck.String
		data.Date = dateInDashboard.String
		return data, "error", nil
	}

	// Try pending table (no problem field)
	err = s.DB.QueryRow(`
		SELECT IFNULL(wo,''), IFNULL(spk,''), IFNULL(teknisi,''),
			IFNULL(type,''), IFNULL(type2,''), IFNULL(sla,''), IFNULL(reason,''),
			IFNULL(tid,''), IFNULL(keterangan,''), IFNULL(mid,''), IFNULL(alamat,''),
			IFNULL(edc_type,''), IFNULL(sn,''), IFNULL(tid_bank,''),
			IFNULL(date_on_check,''), IFNULL(date,''), IFNULL(ta_feedback,'')
		FROM pending WHERE id_task = ?`, idTask,
	).Scan(
		&data.WO, &data.SPK, &data.Teknisi,
		&data.Type, &data.Type2, &data.SLA, &data.Reason,
		&data.TID, &data.Keterangan, &data.Mid, &data.Alamat,
		&data.EdcType, &data.Sn, &data.TID2,
		&dateOnCheck, &dateInDashboard, &data.TaFB,
	)
	if err == nil {
		data.DateCheck = dateOnCheck.String
		data.Date = dateInDashboard.String
		data.Problem = "---"
		return data, "pending", nil
	}

	// Try temp_submission table (includes problem field)
	err = s.DB.QueryRow(`
		SELECT IFNULL(wo,''), IFNULL(spk,''), IFNULL(teknisi,''), IFNULL(problem,''),
			IFNULL(type,''), IFNULL(type2,''), IFNULL(sla,''), IFNULL(reason,''),
			IFNULL(tid,''), IFNULL(keterangan,''), IFNULL(mid,''), IFNULL(alamat,''),
			IFNULL(edc_type,''), IFNULL(sn,''), IFNULL(tid_bank,''),
			IFNULL(date_on_check,''), IFNULL(date,''), IFNULL(ta_feedback,'')
		FROM temp_submission WHERE id_task = ? LIMIT 1`, idTask,
	).Scan(
		&data.WO, &data.SPK, &data.Teknisi, &data.Problem,
		&data.Type, &data.Type2, &data.SLA, &data.Reason,
		&data.TID, &data.Keterangan, &data.Mid, &data.Alamat,
		&data.EdcType, &data.Sn, &data.TID2,
		&dateOnCheck, &dateInDashboard, &data.TaFB,
	)
	if err == nil {
		data.DateCheck = dateOnCheck.String
		data.Date = dateInDashboard.String
		return data, "temp_submission", nil
	}

	return data, "", fmt.Errorf("id_task %s not found in any table", idTask)
}

// UpsertTempSubmission creates or updates a temp_submission record from error/pending data.
func (s *TaskService) UpsertTempSubmission(email, method, id, logEdit string) error {
	var (
		dataWO, dataSPK, dataProblem                                                        sql.NullString
		dataType, dataType2, dataKeterangan, dataDesc                                       sql.NullString
		dataCompany, dataReason, dataTID, dataMerchant                                      sql.NullString
		dataTeknisi, dataMID, dataAlamat, dataTipeEDC, dataSnEdc, dataTIDBank, dataFeedback sql.NullString
		dataReceivedDateSPK, dataSLA, dataTimeStart, dataTimeStop                           sql.NullString
		dataDateInDashboard, dataDateOnCheck                                                sql.NullString
		typeCase                                                                            string
	)

	cols := "wo, spk, receiveDate, type, type2, sla, time_start, time_stop, keterangan, `desc`, company, reason, tid, merchant, teknisi, mid, alamat, edc_type, sn, tid_bank, date, date_on_check, ta_feedback"

	// Try error table first (has problem column)
	err := s.DB.QueryRow(
		"SELECT "+cols+", problem FROM error WHERE id_task = ? LIMIT 1", id,
	).Scan(
		&dataWO, &dataSPK, &dataReceivedDateSPK, &dataType, &dataType2, &dataSLA,
		&dataTimeStart, &dataTimeStop, &dataKeterangan, &dataDesc, &dataCompany, &dataReason,
		&dataTID, &dataMerchant, &dataTeknisi, &dataMID, &dataAlamat, &dataTipeEDC,
		&dataSnEdc, &dataTIDBank, &dataDateInDashboard, &dataDateOnCheck, &dataFeedback, &dataProblem,
	)

	if err == sql.ErrNoRows {
		// Try pending table (no problem column)
		err = s.DB.QueryRow(
			"SELECT "+cols+" FROM pending WHERE id_task = ? LIMIT 1", id,
		).Scan(
			&dataWO, &dataSPK, &dataReceivedDateSPK, &dataType, &dataType2, &dataSLA,
			&dataTimeStart, &dataTimeStop, &dataKeterangan, &dataDesc, &dataCompany, &dataReason,
			&dataTID, &dataMerchant, &dataTeknisi, &dataMID, &dataAlamat, &dataTipeEDC,
			&dataSnEdc, &dataTIDBank, &dataDateInDashboard, &dataDateOnCheck, &dataFeedback,
		)
		if err != nil {
			return fmt.Errorf("upsert temp submission: %w", err)
		}
		typeCase = "pending"
		dataProblem = sql.NullString{String: "---", Valid: true}
	} else if err != nil {
		return fmt.Errorf("upsert temp submission: %w", err)
	} else {
		typeCase = "error"
	}

	// Parse time fields
	parseTime := func(ns sql.NullString) *time.Time {
		if ns.Valid && ns.String != "" {
			if t, err := time.Parse("2006-01-02 15:04:05", ns.String); err == nil {
				return &t
			}
		}
		return nil
	}

	parsedReceivedDateSPK := parseTime(dataReceivedDateSPK)
	parsedSLA := parseTime(dataSLA)
	parsedTimeStart := parseTime(dataTimeStart)
	parsedTimeStop := parseTime(dataTimeStop)
	parsedDateInDashboard := parseTime(dataDateInDashboard)
	parsedDateOnCheck := parseTime(dataDateOnCheck)

	// Check if record already exists
	var count int
	if err := s.DB.QueryRow("SELECT COUNT(*) FROM temp_submission WHERE id_task = ?", id).Scan(&count); err != nil {
		return err
	}

	// Helper: convert sql.NullString to interface{} for DB params
	val := func(ns sql.NullString) interface{} {
		if ns.Valid {
			return ns.String
		}
		return nil
	}

	if count > 0 {
		_, err = s.DB.Exec(`UPDATE temp_submission SET
			wo=?, spk=?, problem=?, received_datetime_spk=?, type_case=?,
			type=?, type2=?, sla=?, time_start=?, time_stop=?,
			keterangan=?, `+"`desc`"+`=?, company=?, reason=?, tid=?,
			merchant=?, teknisi=?, mid=?, alamat=?, edc_type=?,
			sn=?, tid_bank=?, date=?, date_on_check=?, ta_feedback=?,
			email=?, method=?, log_edit=?
			WHERE id_task=?`,
			val(dataWO), val(dataSPK), val(dataProblem), parsedReceivedDateSPK, typeCase,
			val(dataType), val(dataType2), parsedSLA, parsedTimeStart, parsedTimeStop,
			val(dataKeterangan), val(dataDesc), val(dataCompany), val(dataReason), val(dataTID),
			val(dataMerchant), val(dataTeknisi), val(dataMID), val(dataAlamat), val(dataTipeEDC),
			val(dataSnEdc), val(dataTIDBank), parsedDateInDashboard, parsedDateOnCheck, val(dataFeedback),
			email, method, logEdit, id,
		)
	} else {
		_, err = s.DB.Exec(`INSERT INTO temp_submission (
			id_task, wo, spk, problem, received_datetime_spk, type_case,
			type, type2, sla, time_start, time_stop,
			keterangan, `+"`desc`"+`, company, reason, tid,
			merchant, teknisi, mid, alamat, edc_type,
			sn, tid_bank, date, date_on_check, ta_feedback,
			email, method, log_edit
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			id, val(dataWO), val(dataSPK), val(dataProblem), parsedReceivedDateSPK, typeCase,
			val(dataType), val(dataType2), parsedSLA, parsedTimeStart, parsedTimeStop,
			val(dataKeterangan), val(dataDesc), val(dataCompany), val(dataReason), val(dataTID),
			val(dataMerchant), val(dataTeknisi), val(dataMID), val(dataAlamat), val(dataTipeEDC),
			val(dataSnEdc), val(dataTIDBank), parsedDateInDashboard, parsedDateOnCheck, val(dataFeedback),
			email, method, logEdit,
		)
	}

	return err
}
