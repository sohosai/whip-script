package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

var Log_enable = true
var tokyoLocation = time.FixedZone("Asia/Tokyo", 9*60*60)

func ErrorLog(s ...string) {
	if Log_enable {
		fmt.Printf(`
****************************************************
error occured!!
%s
****************************************************
`, strings.Join(s, " "))
	}
}
func Log(s ...string) {
	if Log_enable {
		fmt.Print(strings.Join(s, " "))
	}
}

func BackupKey(path, channelID, key string) error {
	if key == "" {
		return errors.New("encryption key is empty")
	}
	type backupEntry struct {
		Timestamp  string `json:"timestamp"`
		ChannelID  string `json:"channel_id"`
		EncryptKey string `json:"encrypt_key"`
	}
	var entries []backupEntry
	data, err := os.ReadFile(path)
	if err == nil {
		trimmed := strings.TrimSpace(string(data))
		if trimmed != "" {
			if err := json.Unmarshal([]byte(trimmed), &entries); err != nil {
				return fmt.Errorf("failed to parse existing key backup: %w", err)
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	entries = append(entries, backupEntry{
		Timestamp:  time.Now().In(tokyoLocation).Format(time.RFC3339),
		ChannelID:  channelID,
		EncryptKey: key,
	})
	output, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, output, 0600)
}
