package core

import (
	"fmt"
	"io"
	"net/http"

	"github.com/BurntSushi/toml"
)

type RequestPayload struct {
	Target     string
	Body       io.Reader
	Auth_token string
}

func LoadConfig(filepath string) (*Config, error) {
	config := &Config{}
	if _, err := toml.DecodeFile(filepath, config); err != nil {
		return nil, err
	}
	return config, nil
}

func (r RequestPayload) ExecuteImageFluxAPI() ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, "https://live-api.imageflux.jp/", r.Body)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to execute request: %v", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Sora-Target", r.Target)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %v", r.Auth_token))

	client := new(http.Client)
	resp, err := client.Do(req)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to read response body: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to read response body: %v", err)
	}

	return respBody, nil
}
