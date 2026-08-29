package redis

import (
	"context"
	"testing"
)

func TestPing(t *testing.T) {
	client, err := NewClient("localhost:6379")
	if err != nil {
		t.Fatal(err)
	}

	defer client.Close()

	err = client.Ping(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	err = client.Set(context.Background(), "migration:go", "hello")
	if err != nil {
		t.Fatal(err)
	}
}
