package channel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sohosai/whip-script/internal/core"
)

type CreateMultiStreamChannelRequest struct {
	HLS             []core.HlsConfig `json:"hls"`
	EncryptKeyUri   string           `json:"encrypt_key_uri"`
	EventWebhookUrl string           `json:"event_webhook_url"`
}

type CreateChannelResponse struct {
	ChannelID string `json:"channel_id"`
	SoraURL   string `json:"sora_url"`
	Ok        bool   `json:"ok,omitempty"`
	Error     string `json:"error,omitempty"`
}

type DeleteChannelRequest struct {
	ChannelID string `json:"channel_id"`
}

type ListPlaylistURLsReq struct {
	ChannelID string `json:"channel_id"`
}

type ListPlaylistURLsResp struct {
	ChannelID string `json:"channel_id"`
	Ok        bool   `json:"ok,omitempty"`
	Error     string `json:"error,omitempty"`
	HLS       []struct {
		ConnectionID string `json:"connection_id"`
		PlaylistURL  string `json:"playlist_url"`
	} `json:"hls"`
}

func pickLatestURL(resp ListPlaylistURLsResp) (string, bool) {
	for _, hls := range resp.HLS {
		if hls.PlaylistURL != "" {
			return hls.PlaylistURL, true
		}
	}
	return "", false
}

func GetPlaylist(channelID, token string) (string, error) {
	if channelID == "" {
		return "", fmt.Errorf("channel ID is empty")
	}
	if token == "" {
		return "", fmt.Errorf("imageflux token is empty")
	}

	start := time.Now()
	timeout := 30 * time.Second
	backoff := 2 * time.Second
	lastErr := "playlist URL is still empty"

	for {
		if time.Since(start) > timeout {
			return "", fmt.Errorf("failed to get m3u8 TIMEOUT after %s: %s", timeout, lastErr)
		}

		reqBody, err := json.Marshal(ListPlaylistURLsReq{ChannelID: channelID})
		if err != nil {
			return "", fmt.Errorf("failed to marshal ListPlaylistURLs request: %w", err)
		}
		rp := core.RequestPayload{
			Target:     "ImageFlux_20200207.ListPlaylistURLs",
			Body:       bytes.NewBuffer(reqBody),
			Auth_token: token,
		}
		body, err := rp.ExecuteImageFluxAPI()
		if err != nil {
			lastErr = fmt.Sprintf("ListPlaylistURLs request failed: %v", err)
		} else {
			var resp ListPlaylistURLsResp
			if err := json.Unmarshal(body, &resp); err != nil {
				lastErr = fmt.Sprintf("failed to parse ListPlaylistURLs response: %v body=%s", err, string(body))
			} else if resp.Error != "" {
				lastErr = fmt.Sprintf("ListPlaylistURLs returned error: %s", resp.Error)
			} else if u, ok := pickLatestURL(resp); ok {
				core.Log(fmt.Sprintf(`
		プレイリストの取得に成功しました〜〜
		****************************************************
		HLS Playlist URL: %s,
		****************************************************
		`, u))
				return u, nil
			} else {
				lastErr = fmt.Sprintf("playlist_url is empty (hls entries=%d)", len(resp.HLS))
			}
		}

		time.Sleep(backoff)
		if backoff < 3*time.Second {
			backoff += 1 * time.Second
		}
	}
}

func CreateChannels(i core.ImagefluxConfig, reqtoken string) (ChannelId string, SoraURL string) {

	createMultiStreamChannelRequest := CreateMultiStreamChannelRequest{
		HLS:             i.HLS,
		EncryptKeyUri:   i.EncryptKeyUri,
		EventWebhookUrl: i.WebhookURL,
	}

	reqBody, err := json.Marshal(createMultiStreamChannelRequest)
	if err != nil {
		panic(err)
	}

	r := core.RequestPayload{
		Target:     "ImageFlux_20200316.CreateMultistreamChannelWithHLS",
		Body:       bytes.NewBuffer(reqBody),
		Auth_token: reqtoken,
	}

	body, err := r.ExecuteImageFluxAPI()
	if err != nil {
		core.ErrorLog(err.Error())
	}

	var createChannelResponse CreateChannelResponse
	if err := json.Unmarshal(body, &createChannelResponse); err != nil {
		core.ErrorLog(fmt.Sprintf("Failed to unmarshal response body: %v\n", err.Error()))
		core.ErrorLog(fmt.Sprintf("Response body: %s\n", string(body)))
		return
	}

	if createChannelResponse.ChannelID == "" {
		core.ErrorLog("チャンネル作成に失敗しました。", createChannelResponse.Error)
		return
	}

	core.Log(fmt.Sprintf(`
		チャンネルが正常に作成されました。
		****************************************************
		Channel ID: %s,
		Sora URL: %s
		****************************************************
	`, createChannelResponse.ChannelID, createChannelResponse.SoraURL))
	return createChannelResponse.ChannelID, createChannelResponse.SoraURL

}

func DeleteChannel(reqtoken string, channelID string) error {
	if channelID == "" {
		return fmt.Errorf("channel ID is empty")
	}
	deleteChannelRequest := DeleteChannelRequest{
		ChannelID: channelID,
	}

	reqBody, err := json.Marshal(deleteChannelRequest)
	if err != nil {
		panic(err)
	}

	r := core.RequestPayload{
		Target:     "ImageFlux_20180501.DeleteChannel",
		Body:       bytes.NewBuffer(reqBody),
		Auth_token: reqtoken,
	}

	body, err := r.ExecuteImageFluxAPI()
	if err != nil {
		return err
	}

	if len(bytes.TrimSpace(body)) == 0 {
		core.Log("チャンネルを削除しました。\n")
		return nil
	}

	var resp struct {
		Ok    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse delete channel response: %w", err)
	}
	if !resp.Ok {
		if resp.Error != "" {
			return fmt.Errorf("delete channel failed: %s", resp.Error)
		}
		return fmt.Errorf("delete channel failed: response=%s", string(body))
	}

	core.Log("チャンネルを削除しました。\n")
	return nil
}

func DefaultImagefluxConfig(encryptKeyUri, webhookURL string) core.ImagefluxConfig {
	return core.ImagefluxConfig{
		EncryptKeyUri: encryptKeyUri,
		WebhookURL:    webhookURL,
		HLS: []core.HlsConfig{
			{
				DurationSeconds: 1,
				StartTimeOffset: -2,
				Video: struct {
					Width  int `json:"width" toml:"width"`
					Height int `json:"height" toml:"height"`
					FPS    int `json:"fps" toml:"fps"`
					BPS    int `json:"bps" toml:"bps"`
				}{
					Width:  1920,
					Height: 1080,
					FPS:    60,
					BPS:    15000000,
				},
				Audio: struct {
					BPS int `json:"bps" toml:"bps"`
				}{
					BPS: 320000,
				},
			},
			{
				DurationSeconds: 1,
				StartTimeOffset: -2,
				Video: struct {
					Width  int `json:"width" toml:"width"`
					Height int `json:"height" toml:"height"`
					FPS    int `json:"fps" toml:"fps"`
					BPS    int `json:"bps" toml:"bps"`
				}{
					Width:  1280,
					Height: 720,
					FPS:    60,
					BPS:    2500000,
				},
				Audio: struct {
					BPS int `json:"bps" toml:"bps"`
				}{
					BPS: 128000,
				},
			},
			{
				DurationSeconds: 1,
				StartTimeOffset: -2,
				Video: struct {
					Width  int `json:"width" toml:"width"`
					Height int `json:"height" toml:"height"`
					FPS    int `json:"fps" toml:"fps"`
					BPS    int `json:"bps" toml:"bps"`
				}{
					Width:  854,
					Height: 480,
					FPS:    24,
					BPS:    950000,
				},
				Audio: struct {
					BPS int `json:"bps" toml:"bps"`
				}{
					BPS: 96000,
				},
			},
		},
	}
}
