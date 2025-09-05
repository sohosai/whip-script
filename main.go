package main

import (
	"fmt"
	"os"

	"github.com/andreykaipov/goobs"
	"github.com/andreykaipov/goobs/api/requests/config"
	"github.com/andreykaipov/goobs/api/requests/stream"
	"github.com/andreykaipov/goobs/api/typedefs"
	"github.com/sohosai/whip-script/internal/channel"
)

var url = os.Getenv("OBS_WEBSOCKET_URL")
var password = os.Getenv("OBS_WEBSOCKET_PASSWORD")

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

func indexOf(s string, sep byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			return i
		}
	}
	return -1
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

	imagefluxCfg := channel.DefaultImagefluxConfig("", "")

	ChannelId, soraURL := channel.CreateChannels(imagefluxCfg, key)
	if ChannelId == "" {
		panic("Channel ID is empty in the response")
	}

	// Extract host from SoraURL
	parsedURL, err := parseURL(soraURL)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse SoraURL: %v", err))
	}

	WHIP := string("whip_custom")
	res, err := client.Config.SetStreamServiceSettings(&config.SetStreamServiceSettingsParams{
		StreamServiceType: &WHIP,
		StreamServiceSettings: &typedefs.StreamServiceSettings{
			Server: "https://" + parsedURL + "/whip/" + ChannelId,
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

	m3u8Url, err := channel.GetPlaylist(ChannelId, key)
	if err != nil {
		panic(fmt.Errorf("failed to get Playlist: %w", err))
	}
	fmt.Printf("HLS playlist URL: %s\n", m3u8Url)
}
