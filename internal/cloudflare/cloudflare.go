package cloudflare

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/sohosai/whip-script/internal/core"
)

type KvResponse struct {
	Result   interface{}   `json:"result,omitempty"`
	Success  bool          `json:"success"`
	Errors   []interface{} `json:"errors,omitempty"`
	Messages []string      `json:"messages,omitempty"`
}

func PutKv(key string, value string, config *core.Config) error {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%v/storage/kv/namespaces/%v/values/%v", config.Cloudflare.KvAccountID, config.Cloudflare.KvNamespaceID, key)
	kvBody := bytes.NewBufferString(value)
	req, err := http.NewRequest(http.MethodPut, url, kvBody)
	if err != nil {
		core.ErrorLog(err.Error())
		return err
	}
	req.Header.Add("Content-Type", "text/plain")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %v", config.Cloudflare.Token))
	core.Log(fmt.Sprintf(`
	key: %s
	config.Cloudflare.KvAccountID: %s
	config.Cloudflare.KvNamespaceID: %s
	config.Cloudflare.token: %s
		`, key, config.Cloudflare.KvAccountID, config.Cloudflare.KvNamespaceID, config.Cloudflare.Token))
	client := new(http.Client)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		core.ErrorLog(err.Error())
		return err
	}

	// Check non-2xx early
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("cloudflare KV write failed: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var kvResponse KvResponse
	if err := json.Unmarshal(respBody, &kvResponse); err != nil {
		core.ErrorLog("Failed to unmarshal response body: %v\n", err.Error())
		core.ErrorLog("Response body: %s\n", string(respBody))
		return err
	}
	if !kvResponse.Success {
		// Bubble up Cloudflare API error content
		// Note: kvResponse.Errors is []interface{}; stringify for visibility
		enc, _ := json.Marshal(kvResponse)
		return fmt.Errorf("cloudflare KV API returned success=false: %s", string(enc))
	}
	return nil
}
