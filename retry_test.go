package goresilience_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	goresilience "github.com/rickKoch/go-resilience"
)

func TestResilienceRetry(t *testing.T) {
	attempts := atomic.Int32{}
	example_error := errors.New("example_error")
	target := "example_target"
	cfg := goresilience.Config{
		Retries: map[string]goresilience.Retry{
			"example_retry": {
				Duration:   "2s",
				MaxRetries: 3,
			},
		},
		Targets: map[string]goresilience.PolicyNames{
			target: {
				Retry: "example_retry",
			},
		},
	}

	policyProvider, err := goresilience.FromConfig(cfg)
	if err != nil {
		t.Fatalf("failed to create a provider from config: %s", err)
	}

	policy := policyProvider.Policy(target)
	exec := goresilience.NewExecutor(context.Background(), policy)
	_, err = exec(func(ctx context.Context) (any, error) {
		attempts.Add(1)
		return "", example_error
	})

	if err != example_error {
		t.Fatalf("it should've failed with retry error, but exited with: %s", err)
	}

	if attempts.Load() != 4 {
		t.Fatal("it should've retry it 3 times")
	}
}

func TestRetryWithSuccessAfterFailures(t *testing.T) {
	attempts := atomic.Int32{}
	successAfter := int32(2) // Succeed on the 3rd attempt
	target := "success_target"

	cfg := goresilience.Config{
		Retries: map[string]goresilience.Retry{
			"success_retry": {
				Duration:   "100ms",
				MaxRetries: 5,
			},
		},
		Targets: map[string]goresilience.PolicyNames{
			target: {
				Retry: "success_retry",
			},
		},
	}

	policyProvider, err := goresilience.FromConfig(cfg)
	if err != nil {
		t.Fatalf("failed to create a provider from config: %s", err)
	}

	policy := policyProvider.Policy(target)
	exec := goresilience.NewExecutor(context.Background(), policy)

	result, err := exec(func(ctx context.Context) (any, error) {
		current := attempts.Add(1)
		if current <= successAfter {
			return nil, errors.New("temporary failure")
		}
		return "success", nil
	})
	if err != nil {
		t.Fatalf("expected success but got error: %s", err)
	}

	if result != "success" {
		t.Fatalf("expected 'success' but got: %v", result)
	}

	if attempts.Load() != successAfter+1 {
		t.Fatalf("expected %d attempts but got: %d", successAfter+1, attempts.Load())
	}
}

func TestRetryWithZeroMaxRetries(t *testing.T) {
	attempts := atomic.Int32{}
	target := "no_retry_target"

	cfg := goresilience.Config{
		Retries: map[string]goresilience.Retry{
			"no_retry": {
				Duration:   "100ms",
				MaxRetries: 0,
			},
		},
		Targets: map[string]goresilience.PolicyNames{
			target: {
				Retry: "no_retry",
			},
		},
	}

	policyProvider, err := goresilience.FromConfig(cfg)
	if err != nil {
		t.Fatalf("failed to create a provider from config: %s", err)
	}

	policy := policyProvider.Policy(target)
	exec := goresilience.NewExecutor(context.Background(), policy)

	_, err = exec(func(ctx context.Context) (any, error) {
		attempts.Add(1)
		return nil, errors.New("always fails")
	})

	if err == nil {
		t.Fatal("expected error but got none")
	}

	if attempts.Load() != 1 {
		t.Fatalf("expected 1 attempt but got: %d", attempts.Load())
	}
}

func TestRetryWithContextCancellation(t *testing.T) {
	attempts := atomic.Int32{}
	target := "context_cancel_target"

	cfg := goresilience.Config{
		Retries: map[string]goresilience.Retry{
			"long_retry": {
				Duration:   "1s",
				MaxRetries: 10,
			},
		},
		Targets: map[string]goresilience.PolicyNames{
			target: {
				Retry: "long_retry",
			},
		},
	}

	policyProvider, err := goresilience.FromConfig(cfg)
	if err != nil {
		t.Fatalf("failed to create a provider from config: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	policy := policyProvider.Policy(target)
	exec := goresilience.NewExecutor(ctx, policy)

	_, err = exec(func(ctx context.Context) (any, error) {
		attempts.Add(1)
		return nil, errors.New("always fails")
	})

	if err == nil {
		t.Fatal("expected error but got none")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded but got: %s", err)
	}

	// Should have attempted at least once but not all 10 times due to context cancellation
	if attempts.Load() < 1 {
		t.Fatalf("expected at least 1 attempt but got: %d", attempts.Load())
	}
}

func TestRetryWithInvalidDuration(t *testing.T) {
	cfg := goresilience.Config{
		Retries: map[string]goresilience.Retry{
			"invalid_retry": {
				Duration:   "invalid_duration",
				MaxRetries: 3,
			},
		},
		Targets: map[string]goresilience.PolicyNames{
			"test_target": {
				Retry: "invalid_retry",
			},
		},
	}

	_, err := goresilience.FromConfig(cfg)
	if err == nil {
		t.Fatal("expected error for invalid duration but got none")
	}

	if fmt.Sprintf("%s", err) == "" {
		t.Fatal("expected non-empty error message")
	}
	t.Logf("Got expected error: %s", err)
}

func TestRetryWithDifferentDurations(t *testing.T) {
	testCases := []struct {
		name     string
		duration string
		expected time.Duration
	}{
		{"milliseconds", "50ms", 50 * time.Millisecond},
		{"seconds", "100ms", 100 * time.Millisecond}, // Use shorter duration for testing
		{"nanoseconds", "1000000ns", 1 * time.Millisecond},
		{"microseconds", "1000us", 1 * time.Millisecond},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			target := "duration_test_target"
			cfg := goresilience.Config{
				Retries: map[string]goresilience.Retry{
					"duration_retry": {
						Duration:   tc.duration,
						MaxRetries: 1,
					},
				},
				Targets: map[string]goresilience.PolicyNames{
					target: {
						Retry: "duration_retry",
					},
				},
			}

			policyProvider, err := goresilience.FromConfig(cfg)
			if err != nil {
				t.Fatalf("failed to create provider: %s", err)
			}

			policy := policyProvider.Policy(target)
			exec := goresilience.NewExecutor(context.Background(), policy)

			start := time.Now()
			attempts := atomic.Int32{}

			_, err = exec(func(ctx context.Context) (any, error) {
				attempts.Add(1)
				return nil, errors.New("test error")
			})

			duration := time.Since(start)

			if err == nil {
				t.Fatal("expected error but got none")
			}

			if attempts.Load() != 2 {
				t.Fatalf("expected 2 attempts but got: %d", attempts.Load())
			}

			// Allow some tolerance for timing (50% margin)
			minExpected := tc.expected - (tc.expected / 2)
			if duration < minExpected {
				t.Fatalf("expected duration >= %v but got: %v", minExpected, duration)
			}
		})
	}
}

func TestRetryWithSuccessOnFirstAttempt(t *testing.T) {
	attempts := atomic.Int32{}
	target := "immediate_success_target"

	cfg := goresilience.Config{
		Retries: map[string]goresilience.Retry{
			"immediate_success_retry": {
				Duration:   "1s",
				MaxRetries: 5,
			},
		},
		Targets: map[string]goresilience.PolicyNames{
			target: {
				Retry: "immediate_success_retry",
			},
		},
	}

	policyProvider, err := goresilience.FromConfig(cfg)
	if err != nil {
		t.Fatalf("failed to create a provider from config: %s", err)
	}

	policy := policyProvider.Policy(target)
	exec := goresilience.NewExecutor(context.Background(), policy)

	result, err := exec(func(ctx context.Context) (any, error) {
		attempts.Add(1)
		return "immediate success", nil
	})
	if err != nil {
		t.Fatalf("expected success but got error: %s", err)
	}

	if result != "immediate success" {
		t.Fatalf("expected 'immediate success' but got: %v", result)
	}

	if attempts.Load() != 1 {
		t.Fatalf("expected 1 attempt but got: %d", attempts.Load())
	}
}

func TestRetryWithNegativeMaxRetries(t *testing.T) {
	attempts := atomic.Int32{}
	target := "unlimited_retry_target"

	cfg := goresilience.Config{
		Retries: map[string]goresilience.Retry{
			"unlimited_retry": {
				Duration:   "10ms",
				MaxRetries: -1, // Unlimited retries
			},
		},
		Targets: map[string]goresilience.PolicyNames{
			target: {
				Retry: "unlimited_retry",
			},
		},
	}

	policyProvider, err := goresilience.FromConfig(cfg)
	if err != nil {
		t.Fatalf("failed to create a provider from config: %s", err)
	}

	// Use a context with timeout to prevent infinite retries in test
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	policy := policyProvider.Policy(target)
	exec := goresilience.NewExecutor(ctx, policy)

	_, err = exec(func(ctx context.Context) (any, error) {
		current := attempts.Add(1)
		if current >= 5 { // Succeed after 5 attempts to test unlimited behavior
			return "finally succeeded", nil
		}
		return nil, errors.New("keep trying")
	})
	if err != nil {
		t.Fatalf("expected success but got error: %s", err)
	}

	if attempts.Load() < 5 {
		t.Fatalf("expected at least 5 attempts but got: %d", attempts.Load())
	}
}

func TestRetryWithNoTargetConfig(t *testing.T) {
	cfg := goresilience.Config{
		Retries: map[string]goresilience.Retry{
			"example_retry": {
				Duration:   "100ms",
				MaxRetries: 3,
			},
		},
		Targets: map[string]goresilience.PolicyNames{},
	}

	policyProvider, err := goresilience.FromConfig(cfg)
	if err != nil {
		t.Fatalf("failed to create a provider from config: %s", err)
	}

	// Request policy for non-existent target
	policy := policyProvider.Policy("non_existent_target")
	exec := goresilience.NewExecutor(context.Background(), policy)

	attempts := atomic.Int32{}
	_, err = exec(func(ctx context.Context) (any, error) {
		attempts.Add(1)
		return nil, errors.New("test error")
	})

	if err == nil {
		t.Fatal("expected error but got none")
	}

	// Should only attempt once (no retry policy applied)
	if attempts.Load() != 1 {
		t.Fatalf("expected 1 attempt but got: %d", attempts.Load())
	}
}

func TestRetryWithErrorTypes(t *testing.T) {
	attempts := atomic.Int32{}
	target := "error_types_target"

	cfg := goresilience.Config{
		Retries: map[string]goresilience.Retry{
			"error_retry": {
				Duration:   "10ms",
				MaxRetries: 2,
			},
		},
		Targets: map[string]goresilience.PolicyNames{
			target: {
				Retry: "error_retry",
			},
		},
	}

	policyProvider, err := goresilience.FromConfig(cfg)
	if err != nil {
		t.Fatalf("failed to create a provider from config: %s", err)
	}

	policy := policyProvider.Policy(target)
	exec := goresilience.NewExecutor(context.Background(), policy)

	// Test with different error types
	_, err = exec(func(ctx context.Context) (any, error) {
		current := attempts.Add(1)
		switch current {
		case 1:
			return nil, fmt.Errorf("wrapped error: %w", errors.New("inner error"))
		case 2:
			return nil, context.DeadlineExceeded
		default:
			return nil, errors.New("final error")
		}
	})

	if err == nil {
		t.Fatal("expected error but got none")
	}

	if attempts.Load() != 3 {
		t.Fatalf("expected 3 attempts but got: %d", attempts.Load())
	}
}

// Benchmark tests
func BenchmarkRetrySuccess(b *testing.B) {
	cfg := goresilience.Config{
		Retries: map[string]goresilience.Retry{
			"bench_retry": {
				Duration:   "1ms",
				MaxRetries: 3,
			},
		},
		Targets: map[string]goresilience.PolicyNames{
			"bench_target": {
				Retry: "bench_retry",
			},
		},
	}

	policyProvider, _ := goresilience.FromConfig(cfg)
	policy := policyProvider.Policy("bench_target")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		exec := goresilience.NewExecutor(context.Background(), policy)
		_, _ = exec(func(ctx context.Context) (any, error) {
			return "success", nil
		})
	}
}

func BenchmarkRetryFailure(b *testing.B) {
	cfg := goresilience.Config{
		Retries: map[string]goresilience.Retry{
			"bench_retry": {
				Duration:   "1ms",
				MaxRetries: 3,
			},
		},
		Targets: map[string]goresilience.PolicyNames{
			"bench_target": {
				Retry: "bench_retry",
			},
		},
	}

	policyProvider, _ := goresilience.FromConfig(cfg)
	policy := policyProvider.Policy("bench_target")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		exec := goresilience.NewExecutor(context.Background(), policy)
		_, _ = exec(func(ctx context.Context) (any, error) {
			return nil, errors.New("always fails")
		})
	}
}
