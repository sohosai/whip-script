package redis

import (
	"context"
	"fmt"
	"strings"

	redislib "github.com/redis/go-redis/v9"
)

type Client struct {
	inner *redislib.Client
}

func NewClient(addr string) (*Client, error) {
    addr = strings.TrimSpace(addr)
    if addr == "" {
        return nil, fmt.Errorf("redis: アドレスが空です。")
    }
	raw := redislib.NewClient(&redislib.Options{
		Addr: addr,
	})

	client := &Client{
		inner: raw,
	}

	return client, nil
}

func (c *Client) Ping(ctx context.Context) error {
	return c.inner.Ping(ctx).Err()
}

func (c *Client) Close() error {
	return c.inner.Close()
}

func (c *Client) Set(ctx context.Context, key string, value string) error {
	return c.inner.Set(ctx, key, value, 0).Err()
}
