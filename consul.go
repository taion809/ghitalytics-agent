package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
)

func fetchFromConsul(ctx context.Context, repo string) (*time.Time, error) {
	kv := consul.KV()
	var result time.Time

	_ = func() error { return nil }
	t, _, err := kv.Get(fmt.Sprintf("ghitalytics/repo/%s", repo), nil)
	if err != nil {
		if strings.Contains(err.Error(), "connection reset by peer") {
			return fetchFromConsul(ctx, repo)
		}

		return nil, err
	}

	if t == nil {
		return nil, nil
	}

	result, err = time.Parse(time.RFC3339, string(t.Value))
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func updateConsul(ctx context.Context, repo string, since time.Time) error {
	kv := consul.KV()

	_, err := kv.Put(&api.KVPair{
		Key:   fmt.Sprintf("ghitalytics/repo/%s", repo),
		Value: []byte(since.Format(time.RFC3339)),
	}, nil)

	return err
}
