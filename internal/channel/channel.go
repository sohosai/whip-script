package channel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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
	HLS       []struct {
		ConnectionID string `json:"connection_id"`
		PlaylistURL  string `json:"playlist_url"`
	} `json:"hls"`
}

func pickLatestURL(resp ListPlaylistURLsResp) (string, bool) {
	if len(resp.HLS) == 0 {
		return "", false
	}
	if resp.HLS[0].PlaylistURL == "" {
		return "", false
	}
	return resp.HLS[0].PlaylistURL, true
}

func GetPlaylist(channelID, token string) (string, error) {
	start := time.Now()
	timeout := 30 * time.Second
	backoff := 2 * time.Second

	for {
		if time.Since(start) > timeout {
			return "", fmt.Errorf("failed to get m3u8 TIMEOUT")
		}

		reqBody, _ := json.Marshal(map[string]string{"channel_id": channelID})
		rp := core.RequestPayload{
			Target:     "ImageFlux_20200207.ListPlaylistURLs",
			Body:       bytes.NewBuffer(reqBody),
			Auth_token: token,
		}
		body, err := rp.ExecuteImageFluxAPI()
		if err == nil {
			var resp ListPlaylistURLsResp
			if err := json.Unmarshal(body, &resp); err == nil {
				if u, ok := pickLatestURL(resp); ok {
					core.Log(fmt.Sprintf(`
		プレイリストの取得に成功しました〜〜
		****************************************************
		HLS Playlist URL: %s,
		****************************************************
		`, u))
					return u, nil
				}
			}
		}

		time.Sleep(backoff)
		if backoff < 3*time.Second {
			backoff += 1 * time.Second
		}
	}
}

func GetKey(urlString string, authToken string) (string, string, error) {
	req, err := http.NewRequest(http.MethodGet, urlString, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/vnd.apple.mpegurl")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		core.ErrorLog(err.Error())
	}

	var indexURL string
	for _, line := range strings.Split(string(body), "\n") {
		if strings.HasPrefix(line, "http") {
			indexURL = strings.Trim(line, "\r\n ")
			break
		}
	}
	core.Log(fmt.Sprintf(`
****************************************************
indexUrl: %v
****************************************************

`, indexURL))

	req, err = http.NewRequest(http.MethodGet, indexURL, nil)
	if err != nil {
		core.ErrorLog(err.Error())
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/vnd.apple.mpegurl")

	res, err = client.Do(req)
	if err != nil {
		core.ErrorLog(err.Error())
		return "", "", err
	}
	defer res.Body.Close()

	body, err = io.ReadAll(res.Body)
	if err != nil {
		core.ErrorLog(err.Error())
		return "", "", err
	}
	dataForKid := string(body)

	lines := strings.Split(dataForKid, "\n")
	var kid string
	for _, line := range lines {
		if strings.HasPrefix(line, "#EXT-X-KEY") {
			rawURI := strings.Split(strings.Split(line, `URI="`)[1], `"`)[0]
			parsedURL, err := url.Parse(rawURI)
			if err != nil {
				return "", "", err
			}
			kid = parsedURL.Query().Get("kid")
			break
		}
	}

	if kid == "" {
		return "", "", nil
	}

	keyRequestBody, err := json.Marshal(map[string]string{"kid": kid})
	if err != nil {
		return "", "", err
	}
	r := core.RequestPayload{
		Target:     "ImageFlux_20200707.GetEncryptKey",
		Body:       bytes.NewBuffer(keyRequestBody),
		Auth_token: authToken,
	}

	body, err = r.ExecuteImageFluxAPI()
	if err != nil {
		return "", "", err
	}

	type EncryptKeyResponse struct {
		EncryptKey string `json:"encrypt_key"`
	}
	var keyResponse EncryptKeyResponse
	if err := json.Unmarshal(body, &keyResponse); err != nil {
		return "", "", err
	}

	return keyResponse.EncryptKey, indexURL, nil
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
