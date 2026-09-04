package ingest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type StreamCredentials struct {
	ChannelID     string `json:"channel_id"`
	LiveURL       string `json:"live_url"`
	EncryptionKey string `json:"encryption_key"`
}

func Send(
	ctx context.Context,
	endpoint string,
	token string,
	credentials StreamCredentials,
) error {
	body, err := json.Marshal(credentials)
	if err != nil {
		return fmt.Errorf("encode stream credentials: %w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("build stream request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("send stream request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("stream credentials request failed: status=%d", response.StatusCode)
	}

	return nil
}
