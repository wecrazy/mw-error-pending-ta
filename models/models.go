package models

// ImageFields lists all photo/image field names used throughout the application.
var ImageFields = []string{
	"x_foto_bast", "x_foto_ceklis", "x_foto_edc", "x_foto_pic",
	"x_foto_setting", "x_foto_thermal", "x_foto_toko", "x_foto_training",
	"x_foto_transaksi", "x_tanda_tangan_pic", "x_tanda_tangan_teknisi",
	"x_foto_sticker_edc", "x_foto_screen_guard", "x_foto_all_transaction",
	"x_foto_transaksi_bmri", "x_foto_transaksi_bni", "x_foto_transaksi_bri",
	"x_foto_transaksi_btn", "x_foto_transaksi_patch", "x_foto_screen_p2g",
	"x_foto_kontak_stiker_pic", "x_foto_selfie_teknisi_merchant",
	"x_foto_selfie_video_call",
}

// --- Request types ---

type RequestData struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	IDTask   string `json:"id_task"`
	Reason   string `json:"reason"`
	KeepData bool   `json:"keep_data"`
	IsPaid   bool   `json:"is_paid"`
}

type RequestDataReason struct {
	Company string `json:"company"`
}

type RequestDataJSON struct {
	IDTask string             `json:"id_task"`
	Data   []RequestDataField `json:"data"`
}

type RequestDataField struct {
	Name      string `json:"name"`
	ArrayType string `json:"arrayType"`
	Index     int    `json:"index"`
	Type      string `json:"type"` // string, boolean, integer, array
}

// --- Response types ---

type DataTablesResponse struct {
	Data []QueryResult `json:"data"`
}

type ResponseDataJSON struct {
	IDTask string           `json:"id_task"`
	Result []DataFieldValue `json:"result"`
}

type DataFieldValue struct {
	Name string      `json:"name"`
	Data interface{} `json:"data"`
}

// --- Data types ---

type QueryResult struct {
	IDTask    string `json:"id_task"`
	TimeStart string `json:"time_start"`
	TimeStop  string `json:"time_stop"`
	TID       string `json:"tid"`
	Teknisi   string `json:"teknisi"`
}

type ErrorData struct {
	WO         string
	SPK        string
	Teknisi    string
	Problem    string
	Type       string
	Type2      string
	SLA        string
	Reason     string
	TID        string
	Keterangan string
	TID2       string
	EdcType    string
	Sn         string
	Alamat     string
	Mid        string
	DateCheck  string
	Date       string
	TaFB       string
}

type JSONFile struct {
	Params map[string]interface{} `json:"params"`
}

// --- Odoo types ---

type Credentials struct {
	JSONRPC string `json:"jsonrpc"`
	Params  struct {
		DB       string `json:"db"`
		Login    string `json:"login"`
		Password string `json:"password"`
	} `json:"params"`
}

type OdooResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  struct {
		Status   int    `json:"status"`
		Success  bool   `json:"success"`
		Response bool   `json:"response"`
		Message  string `json:"message"`
	} `json:"result"`
}

type OdooReasonResponse struct {
	Result []struct {
		ID          int           `json:"id"`
		XName       string        `json:"x_name"`
		XCompanyID  []interface{} `json:"x_company_id"`
		XReasonCode string        `json:"x_reason_code"`
	} `json:"result"`
}

type OdooStageResponse struct {
	Result []struct {
		StageID []interface{} `json:"stage_id"`
	} `json:"result"`
}

// --- Task data (incoming from external service) ---

type TaskData struct {
	JSONRPC string         `json:"jsonrpc"`
	Params  TaskDataParams `json:"params"`
}

type TaskDataParams struct {
	ID                       string `json:"id"`
	Model                    string `json:"model"`
	TimesheetTimerFirstStart string `json:"timesheet_timer_first_start"`
	TimesheetTimerLastStop   string `json:"timesheet_timer_last_stop"`

	// Photo & signature fields
	XFotoBast            string `json:"x_foto_bast"`
	XFotoCeklis          string `json:"x_foto_ceklis"`
	XFotoEdc             string `json:"x_foto_edc"`
	XFotoPic             string `json:"x_foto_pic"`
	XFotoSetting         string `json:"x_foto_setting"`
	XFotoToko            string `json:"x_foto_toko"`
	XFotoTransaksi       string `json:"x_foto_transaksi"`
	XTandaTanganPic      string `json:"x_tanda_tangan_pic"`
	XTandaTanganTeknisi  string `json:"x_tanda_tangan_teknisi"`
	XFotoTraining        string `json:"x_foto_training"`
	XFotoThermal         string `json:"x_foto_thermal"`
	XFotoStickerEdc      string `json:"x_foto_sticker_edc"`
	XFotoScreenGuard     string `json:"x_foto_screen_guard"`
	XFotoAllTransaction  string `json:"x_foto_all_transaction"`
	XFotoTransaksiBmri   string `json:"x_foto_transaksi_bmri"`
	XFotoTransaksiBni    string `json:"x_foto_transaksi_bni"`
	XFotoTransaksiBri    string `json:"x_foto_transaksi_bri"`
	XFotoTransaksiBtn    string `json:"x_foto_transaksi_btn"`
	XFotoTransaksiPatch  string `json:"x_foto_transaksi_patch"`
	XFotoKontakStikerPic string `json:"x_foto_kontak_stiker_pic"`
	XFotoScreenP2g       string `json:"x_foto_screen_p2g"`
	XFotoSelfieTeknisi   string `json:"x_foto_selfie_teknisi_merchant"`
	XFotoSelfieVideoCall string `json:"x_foto_selfie_video_call"`

	// Meta fields
	CompanyID      int    `json:"company_id"`
	XLat           string `json:"x_latitude"`
	XLong          string `json:"x_longitude"`
	XReasonCodeID  int    `json:"x_reason_code_id"`
	XKeterangan    string `json:"x_keterangan"`
	XSupplyThermal int    `json:"x_supply_thermal"`
}

// ImageMap returns a mapping of field name → base64 image data.
// Replaces the massive if-else chains in the original code.
func (p TaskDataParams) ImageMap() map[string]string {
	return map[string]string{
		"x_foto_bast":                    p.XFotoBast,
		"x_foto_ceklis":                  p.XFotoCeklis,
		"x_foto_edc":                     p.XFotoEdc,
		"x_foto_pic":                     p.XFotoPic,
		"x_foto_setting":                 p.XFotoSetting,
		"x_foto_toko":                    p.XFotoToko,
		"x_foto_transaksi":               p.XFotoTransaksi,
		"x_tanda_tangan_pic":             p.XTandaTanganPic,
		"x_tanda_tangan_teknisi":         p.XTandaTanganTeknisi,
		"x_foto_training":                p.XFotoTraining,
		"x_foto_thermal":                 p.XFotoThermal,
		"x_foto_sticker_edc":             p.XFotoStickerEdc,
		"x_foto_screen_guard":            p.XFotoScreenGuard,
		"x_foto_all_transaction":         p.XFotoAllTransaction,
		"x_foto_transaksi_bmri":          p.XFotoTransaksiBmri,
		"x_foto_transaksi_bni":           p.XFotoTransaksiBni,
		"x_foto_transaksi_bri":           p.XFotoTransaksiBri,
		"x_foto_transaksi_btn":           p.XFotoTransaksiBtn,
		"x_foto_transaksi_patch":         p.XFotoTransaksiPatch,
		"x_foto_screen_p2g":              p.XFotoScreenP2g,
		"x_foto_kontak_stiker_pic":       p.XFotoKontakStikerPic,
		"x_foto_selfie_teknisi_merchant": p.XFotoSelfieTeknisi,
		"x_foto_selfie_video_call":       p.XFotoSelfieVideoCall,
	}
}
