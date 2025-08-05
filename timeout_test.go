package goresilience_test

import (
	"context"
	"testing"
	"time"

	goresilience "github.com/rickKoch/go-resilience"
)

var timeouts = map[string]string{
	"long":  "10s",
	"mid":   "8s",
	"short": "1s",
}

func TestResilienceTimeout(t *testing.T) {
	target := "example_target"
	cfg := goresilience.Config{
		Timeouts: timeouts,
		Targets: map[string]goresilience.PolicyNames{
			"example_target": {
				Timeout: "short",
			},
		},
	}

	policyProvider, err := goresilience.FromConfig(cfg)
	if err != nil {
		t.Fatalf("failed to create a provider from config: %s", err)
	}

	policy := policyProvider.Policy(target)
	exec := goresilience.NewExecWithPolicy(context.Background(), policy)
	_, err = exec(func(ctx context.Context) (any, error) {
		time.Sleep(2 * time.Second)
		return "", nil
	})
	if err != context.DeadlineExceeded {
		t.Fatalf("it should've failed with timeout error, but exited with: %s", err)
	}
}
