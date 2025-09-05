package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/andreykaipov/goobs"
	"github.com/andreykaipov/goobs/api/requests/config"
	"github.com/andreykaipov/goobs/api/requests/stream"
	"github.com/andreykaipov/goobs/api/typedefs"
)

var url = os.Getenv("OBS_WEBSOCKET_URL")
var password = os.Getenv("OBS_WEBSOCKET_PASSWORD")

type CreateMultiStreamChannelRequest struct {
	HLS             []HlsConfig `json:"hls"`
	EncryptKeyUri   string      `json:"encrypt_key_uri"`
	EventWebhookUrl string      `json:"event_webhook_url"`
}
type RequestPayload struct {
	Target     string
	Body       io.Reader
	Auth_token string
}
type CreateChannelResponse struct {
	ChannelID string `json:"channel_id"`
	SoraURL   string `json:"sora_url"`
}
type HlsConfig struct {
	DurationSeconds int `json:"durationSeconds" toml:"duration-seconds"`
	StartTimeOffset int `json:"startTimeOffset" toml:"start-time-offset"`
	Video           struct {
		Width  int `json:"width" toml:"width"`
		Height int `json:"height" toml:"height"`
		FPS    int `json:"fps" toml:"fps"`
		BPS    int `json:"bps" toml:"bps"`
	} `json:"video" toml:"video"`
	Audio struct {
		BPS int `json:"bps" toml:"bps"`
	} `json:"audio" toml:"audio"`
	Archive struct {
		ArchiveDestinationId string `json:"archive_destination_id" toml:"archive_destination_id"`
	} `json:"archive" toml:"archive"`
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
func parseURL(soraURL string) (string, error) {
	// Remove protocol (e.g., wss://)
	withoutProto := soraURL
	if idx := len("wss://"); len(soraURL) > idx && soraURL[:idx] == "wss://" {
		withoutProto = soraURL[idx:]
	} else if idx := len("https://"); len(soraURL) > idx && soraURL[:idx] == "https://" {
		withoutProto = soraURL[idx:]
	}
	// Extract host (up to first slash or end)
	host := withoutProto
	if idx := indexOf(host, '/'); idx != -1 {
		host = host[:idx]
	}
	return host, nil
}

// indexOf returns the index of the first occurrence of sep in s, or -1 if not found.
func indexOf(s string, sep byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			return i
		}
	}
	return -1
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
		rp := RequestPayload{
			Target:     "ImageFlux_20200207.ListPlaylistURLs",
			Body:       bytes.NewBuffer(reqBody),
			Auth_token: token,
		}
		body, err := rp.ExecuteImageFluxAPI()
		if err == nil {
			var resp ListPlaylistURLsResp
			if err := json.Unmarshal(body, &resp); err == nil {
				if u, ok := pickLatestURL(resp); ok {
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

func main() {
	client, err := goobs.New(url, goobs.WithPassword(password))
	if err != nil {
		panic(err)
	}
	defer client.Disconnect()
	var key = os.Getenv("IMAGE_FLUX_AUTH_KEY")
	if key == "" {
		panic("IMAGE_FLUX_AUTH_KEY environment variable is not set")
	}

	reqBody, err := json.Marshal(CreateMultiStreamChannelRequest{
		HLS: []HlsConfig{
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
	})
	if err != nil {
		panic(err)
	}

	r := RequestPayload{
		Target:     "ImageFlux_20200316.CreateMultistreamChannelWithHLS",
		Body:       bytes.NewBuffer(reqBody),
		Auth_token: key,
	}

	body, err := r.ExecuteImageFluxAPI()
	if err != nil {
		panic(fmt.Sprintf("Failed to execute ImageFlux API: %v", err))
	}

	var createChannelResponse CreateChannelResponse
	if err := json.Unmarshal(body, &createChannelResponse); err != nil {
		panic(fmt.Sprintf("Failed to unmarshal response body: %v\n", err.Error()))
	}
	fmt.Printf("CreateChannelResponse: %+v\n", createChannelResponse)
	token := createChannelResponse.ChannelID
	if token == "" {
		panic("Channel ID is empty in the response")
	}
	// Extract host from SoraURL
	soraURL := createChannelResponse.SoraURL
	parsedURL, err := parseURL(soraURL)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse SoraURL: %v", err))
	}

	version, err := client.General.GetVersion()
	if err != nil {
		panic(err)
	}
	WHIP := string("whip_custom")
	res, err := client.Config.SetStreamServiceSettings(&config.SetStreamServiceSettingsParams{
		StreamServiceType: &WHIP,
		StreamServiceSettings: &typedefs.StreamServiceSettings{
			Server: "https://" + parsedURL + "/whip/" + token,
		},
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("SetStreamServiceSettings response: %+v\n", res)

	res2, err := client.Stream.StartStream(&stream.StartStreamParams{})
	if err != nil {
		panic(err)
	}

	fmt.Printf("StartStream response: %+v\n", res2)

	m3u8Url, err := GetPlaylist(token, key)
	if err != nil {
		panic(fmt.Errorf("failed to get Playlist: %w", err))
	}
	fmt.Printf("HLS playlist URL: %s\n", m3u8Url)

	fmt.Printf("OBS Studio version: %s\n", version.ObsVersion)
	fmt.Printf("Server protocol version: %s\n", version.ObsWebSocketVersion)
	fmt.Printf("Client protocol version: %s\n", goobs.ProtocolVersion)
	fmt.Printf("Client library version: %s\n", goobs.LibraryVersion)
}
