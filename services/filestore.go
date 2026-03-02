package services

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"middleware-pending-error-ta/models"
)

// imageFieldSet is a pre-built set for fast lookup of image field names.
var imageFieldSet map[string]bool

func init() {
	imageFieldSet = make(map[string]bool, len(models.ImageFields))
	for _, f := range models.ImageFields {
		imageFieldSet[f] = true
	}
}

// PostToFileStore sends JSON data to the file store service.
func PostToFileStore(url string, data string) (int, error) {
	resp, err := http.Post(url, "application/json", bytes.NewBufferString(data))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body) // drain body

	return resp.StatusCode, nil
}

// DeleteFolder removes a directory and all its contents.
func DeleteFolder(folderPath string) error {
	absPath, err := filepath.Abs(folderPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	info, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("folder not found: %s", absPath)
	} else if err != nil {
		return fmt.Errorf("failed to stat folder: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", absPath)
	}

	return os.RemoveAll(absPath)
}

// RemoveFalsyValues removes image/signature fields and any falsy values
// (empty strings, false, zero) from a JSON map. Operates recursively.
func RemoveFalsyValues(data map[string]interface{}) {
	for k, v := range data {
		// Remove all image/signature keys
		if imageFieldSet[k] {
			delete(data, k)
			continue
		}

		switch val := v.(type) {
		case map[string]interface{}:
			RemoveFalsyValues(val)
			if len(val) == 0 {
				delete(data, k)
			}
		case bool:
			if !val {
				delete(data, k)
			}
		case string:
			if val == "" {
				delete(data, k)
			}
		case float64:
			if val == 0 {
				delete(data, k)
			}
		}
	}
}
