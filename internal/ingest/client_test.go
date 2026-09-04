package ingest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSend(t *testing.T) {
	t.Parallel()

	want := StreamCredentials{
		ChannelID:     "uni",
		LiveURL:       "https://example.invalid/live.m3u8",
		EncryptionKey: "0011223344556677",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want %q", r.Method, http.MethodPost)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want %q", got, "application/json")
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test-token")
		}

		var got StreamCredentials
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if got != want {
			t.Errorf("credentials = %#v, want %#v", got, want)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	if err := Send(context.Background(), server.URL, "test-token", want); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
}

func TestSendRejectsUnexpectedStatus(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "sensitive response body", http.StatusInternalServerError)
	}))
	defer server.Close()

	err := Send(context.Background(), server.URL, "test-token", StreamCredentials{
		ChannelID:     "uni",
		LiveURL:       "https://example.invalid/live.m3u8",
		EncryptionKey: "0011223344556677",
	})
	if err == nil {
		t.Fatal("Send() error = nil, want non-nil")
	}
	if got, want := err.Error(), "stream credentials request failed: status=500"; got != want {
		t.Fatalf("Send() error = %q, want %q", got, want)
	}
	if strings.Contains(err.Error(), "sensitive response body") {
		t.Fatal("Send() error contains response body")
	}
}
