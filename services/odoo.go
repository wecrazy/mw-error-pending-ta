package services

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"middleware-pending-error-ta/config"
	"middleware-pending-error-ta/models"
)

// OdooClient handles all communication with the Odoo API.
type OdooClient struct {
	Config *config.Config
}

// NewOdooClient creates a new Odoo client.
func NewOdooClient(cfg *config.Config) *OdooClient {
	return &OdooClient{Config: cfg}
}

func (o *OdooClient) httpClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
}

// Login authenticates against Odoo and returns cookies for subsequent requests.
func (o *OdooClient) Login(user, pass string) (bool, []*http.Cookie, error) {
	if user == "" {
		return false, nil, nil
	}

	creds := models.Credentials{JSONRPC: "2.0"}
	creds.Params.DB = "gsa_db"
	creds.Params.Login = user
	creds.Params.Password = pass

	payload, err := json.Marshal(creds)
	if err != nil {
		return false, nil, err
	}

	client := o.httpClient()
	req, err := http.NewRequest("POST", o.Config.OdooLoginURL, bytes.NewBuffer(payload))
	if err != nil {
		return false, nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return false, nil, err
	}
	defer resp.Body.Close()

	cookies := resp.Cookies()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, nil, err
	}

	var parsed struct {
		Result struct {
			Username string `json:"username"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return false, nil, fmt.Errorf("error parsing odoo login response: %w", err)
	}

	return parsed.Result.Username == user, cookies, nil
}

// Call sends a JSON-RPC request to an Odoo endpoint with session cookies.
func (o *OdooClient) Call(taskURL string, data []byte, cookies []*http.Cookie) (string, error) {
	client := o.httpClient()

	jar, _ := cookiejar.New(nil)
	parsedURL, err := url.Parse(taskURL)
	if err != nil {
		return "", err
	}
	if cookies != nil {
		jar.SetCookies(parsedURL, cookies)
	}
	client.Jar = jar

	resp, err := client.Post(taskURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	log.Printf("Odoo response from %s: %s", taskURL, string(body))
	return string(body), nil
}

// GetStage fetches the current stage of a task from Odoo.
func (o *OdooClient) GetStage(taskID string, cookies []*http.Cookie) (string, error) {
	body := fmt.Sprintf(
		`{"jsonrpc":"2.0","params":{"company_id":3,"model":"project.task","fields":["stage_id"],"domain":[["id","=",%s]],"order":"create_date asc"}}`,
		taskID,
	)

	ret, err := o.Call(o.Config.OdooGetURL, []byte(body), cookies)
	if err != nil {
		return "", fmt.Errorf("failed to fetch stage from odoo: %w", err)
	}

	var response models.OdooStageResponse
	if err := json.Unmarshal([]byte(ret), &response); err != nil {
		return "", fmt.Errorf("invalid odoo stage response: %w", err)
	}

	if len(response.Result) < 1 {
		return "", fmt.Errorf("id_task not found in odoo")
	}

	stage, ok := response.Result[0].StageID[1].(string)
	if !ok {
		return "", fmt.Errorf("odoo stage_id is null or invalid")
	}

	return stage, nil
}

// PostUpdate sends updated task data to Odoo and checks for success.
func (o *OdooClient) PostUpdate(data []byte, cookies []*http.Cookie) error {
	ret, err := o.Call(o.Config.OdooUpdateURL, data, cookies)
	if err != nil {
		return fmt.Errorf("failed to post update to odoo: %w", err)
	}

	var resp models.OdooResponse
	if err := json.Unmarshal([]byte(ret), &resp); err != nil {
		return fmt.Errorf("invalid odoo update response: %w", err)
	}

	if resp.Result.Message != "Success" {
		return fmt.Errorf("odoo update failed: %s", resp.Result.Message)
	}

	return nil
}
