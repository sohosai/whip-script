package firestore

import (
	"context"
	"log"
	"sync"

	gfirestore "cloud.google.com/go/firestore"
	"github.com/sohosai/whip-script/internal/core"
)

var (
	clientMu sync.RWMutex
	client   *gfirestore.Client
)

func SetClient(c *gfirestore.Client) {
	clientMu.Lock()
	defer clientMu.Unlock()
	client = c
}

func Client() *gfirestore.Client {
	clientMu.RLock()
	defer clientMu.RUnlock()
	return client
}

// Test writes a sample user document for connectivity verification.
func Write(value string, key string) {
	ctx := context.Background()
	_, _, err := client.Collection("default").Add(ctx, map[string]interface{}{
		key: value,
	})

	if err != nil {
		log.Fatalf("Failed adding alovelace: %v", err)
	} else {
		core.Log("m3u8をfirestoreに書き込みました")
	}
}
