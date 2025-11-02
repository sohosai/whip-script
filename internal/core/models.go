package core

type Config struct {
	Imageflux  ImagefluxConfig  `toml:"imageflux"`
	Cloudflare CloudflareConfig `toml:"cloudflare"`
	Patlite    PatliteConfig    `toml:"patlite"`
	Firebase   FirebaseConfig   `toml:"firebase"`
}

type ImagefluxConfig struct {
	// Encrypt Key URI
	EncryptKeyUri string `toml:"encrypt-key-uri"`
	// API token for ImageFlux
	Token string `toml:"token"`
	// HLS configuration
	HLS []HlsConfig `toml:"hls"`
	// Webhook URL
	WebhookURL string `json:"event-webhook-url" toml:"event_webhook_url"`
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

type CloudflareConfig struct {
	// API token for Cloudflare
	Token string `toml:"token"`
	// Cloudflare KV Account ID
	KvAccountID string `toml:"kv-account-id"`
	// Cloudflare KV Namespace ID
	KvNamespaceID string `toml:"kv-namespace-id"`
}

type HLS struct {
	ConnectionID string `json:"connection_id"`
	PlaylistURL  string `json:"playlist_url"`
}

type ListChannelResponse struct {
	ChannelID string `json:"channel_id"`
	HLS       []HLS  `json:"hls"`
}

type PatliteConfig struct {
	IP string `toml:"IP"`
}

type FirebaseConfig struct {
	CredentialPath string `toml:"credentialpath"`
}
