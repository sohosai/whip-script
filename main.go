package main

import (
	"fmt"
	"log"
	"os"

	"github.com/andreykaipov/goobs"
	obsconfig "github.com/andreykaipov/goobs/api/requests/config"
	"github.com/andreykaipov/goobs/api/requests/stream"
	"github.com/andreykaipov/goobs/api/typedefs"
	"github.com/sohosai/whip-script/internal/channel"
	"github.com/sohosai/whip-script/internal/core"
	"github.com/urfave/cli/v2"
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
	cfg := &core.Config{}
	patliteEnabled := true

	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   "config.toml",
				Usage:   "Load configuration from `FILE`",
			},
			&cli.BoolFlag{
				Name:  "nolog",
				Usage: "Disable logging",
			},
			&cli.BoolFlag{
				Name:  "nolite",
				Usage: "Disable patlite",
			},
		},
		Action: func(c *cli.Context) error {
			if configPath := c.String("config"); configPath != "" {
				loaded, err := core.LoadConfig(configPath)
				if err != nil {
					core.ErrorLog("Failed to load config file: ", err.Error())
					return err
				}
				cfg = loaded
			}

			if c.Bool("nolog") {
				core.Log_enable = false
			}
			if c.Bool("nolite") {
				patliteEnabled = false
			}
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

	_ = patliteEnabled

	client, err := goobs.New(url, goobs.WithPassword(password))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect()

	key := cfg.Imageflux.Token
	if key == "" {
		log.Fatal("ImageFlux token is empty")
	}

	ChannelId, soraURL := channel.CreateChannels(cfg.Imageflux, key)
	if ChannelId == "" {
		log.Fatal("channel ID is empty in the response")
	}

	parsedURL, err := parseURL(soraURL)
	if err != nil {
		log.Fatalf("failed to parse SoraURL: %v", err)
	}

	WHIP := string("whip_custom")
	res, err := client.Config.SetStreamServiceSettings(&obsconfig.SetStreamServiceSettingsParams{
		StreamServiceType: &WHIP,
		StreamServiceSettings: &typedefs.StreamServiceSettings{
			Server: "https://" + parsedURL + "/whip/" + ChannelId,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("SetStreamServiceSettings response: %+v\n", res)

	res2, err := client.Stream.StartStream(&stream.StartStreamParams{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("StartStream response: %+v\n", res2)

	m3u8Url, err := channel.GetPlaylist(ChannelId, key)
	if err != nil {
		log.Fatalf("failed to get Playlist: %v", err)
	}
	fmt.Printf("HLS playlist URL: %s\n", m3u8Url)
}
