package goresilience_test

import (
	"context"
	"errors"
	"testing"
	"time"

	goresilience "github.com/rickKoch/go-resilience"
)

var (
	testError     = errors.New("test error")
	successResult = "success"
)

func TestCircuitBreakerBasicFunctionality(t *testing.T) {
	target := "test_target"
	cfg := goresilience.Config{
		CircuitBreakers: map[string]goresilience.CircuitBreaker{
			"test_cb": {
				MaxRequests: 3,
				Interval:    "10s",
				Timeout:     "5s",
				Failures:    2, // Trip after 2 failures
			},
		},
		Targets: map[string]goresilience.PolicyNames{
			target: {
				CircuitBreaker: "test_cb",
			},
		},
	}

	provider, err := goresilience.FromConfig(cfg)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	policy := provider.Policy(target)
	exec := goresilience.NewExecWithPolicy(context.Background(), policy)

	// Test successful operation
	result, err := exec(func(ctx context.Context) (any, error) {
		return successResult, nil
	})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if result != successResult {
		t.Fatalf("expected %s, got %v", successResult, result)
	}
}

func TestCircuitBreakerTripping(t *testing.T) {
	target := "test_target"
	cfg := goresilience.Config{
		CircuitBreakers: map[string]goresilience.CircuitBreaker{
			"test_cb": {
				MaxRequests: 1,
				Interval:    "10s",
				Timeout:     "2s", // Short timeout for faster test
				Failures:    2,    // Trip after 2 failures
			},
		},
		Targets: map[string]goresilience.PolicyNames{
			target: {
				CircuitBreaker: "test_cb",
			},
		},
	}

	provider, err := goresilience.FromConfig(cfg)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	policy := provider.Policy(target)
	exec := goresilience.NewExecWithPolicy(context.Background(), policy)

	// Cause failures to trip the circuit breaker
	for i := 0; i < 3; i++ {
		_, err := exec(func(ctx context.Context) (any, error) {
			return nil, testError
		})
		if err != testError {
			t.Logf("Attempt %d: got error %v (expected %v)", i+1, err, testError)
		}
	}

	// Now the circuit should be open - verify it fails fast
	_, err = exec(func(ctx context.Context) (any, error) {
		t.Error("operation should not be executed when circuit is open")
		return successResult, nil
	})

	if err != goresilience.ErrOpenState {
		t.Fatalf("expected ErrOpenState, got: %v", err)
	}
}

func TestCircuitBreakerRecovery(t *testing.T) {
	target := "test_target"
	cfg := goresilience.Config{
		CircuitBreakers: map[string]goresilience.CircuitBreaker{
			"test_cb": {
				MaxRequests: 2,
				Interval:    "100ms",
				Timeout:     "500ms", // Short timeout for faster test
				Failures:    2,       // Trip after 2 failures
			},
		},
		Targets: map[string]goresilience.PolicyNames{
			target: {
				CircuitBreaker: "test_cb",
			},
		},
	}

	provider, err := goresilience.FromConfig(cfg)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	policy := provider.Policy(target)
	exec := goresilience.NewExecWithPolicy(context.Background(), policy)

	// Trip the circuit breaker
	_, err = exec(func(ctx context.Context) (any, error) {
		return nil, testError
	})
	if err != testError {
		t.Fatalf("expected test error, got: %v", err)
	}

	// Another failure to ensure it's tripped
	_, err = exec(func(ctx context.Context) (any, error) {
		return nil, testError
	})
	if err != testError {
		t.Fatalf("expected test error, got: %v", err)
	}

	// Verify circuit is open
	_, err = exec(func(ctx context.Context) (any, error) {
		t.Error("operation should not be executed when circuit is open")
		return nil, nil
	})
	if err != goresilience.ErrOpenState {
		t.Fatalf("expected ErrOpenState, got: %v", err)
	}

	// Wait for circuit to move to half-open state
	time.Sleep(600 * time.Millisecond)

	// Test recovery with successful operation
	result, err := exec(func(ctx context.Context) (any, error) {
		return successResult, nil
	})
	if err != nil {
		t.Fatalf("expected success during recovery, got error: %v", err)
	}
	if result != successResult {
		t.Fatalf("expected %s, got %v", successResult, result)
	}
}

func TestIsErrorPermanent(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "ErrOpenState should be permanent",
			err:      goresilience.ErrOpenState,
			expected: true,
		},
		{
			name:     "ErrTooManyRequests should be permanent",
			err:      goresilience.ErrTooManyRequests,
			expected: true,
		},
		{
			name:     "Regular error should not be permanent",
			err:      testError,
			expected: false,
		},
		{
			name:     "Nil error should not be permanent",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := goresilience.IsErrorPermanent(tt.err)
			if result != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCircuitBreakerConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		config      goresilience.CircuitBreaker
		expectError bool
	}{
		{
			name: "valid configuration",
			config: goresilience.CircuitBreaker{
				MaxRequests: 5,
				Interval:    "10s",
				Timeout:     "30s",
				Failures:    3,
			},
			expectError: false,
		},
		{
			name: "invalid interval duration",
			config: goresilience.CircuitBreaker{
				MaxRequests: 5,
				Interval:    "invalid",
				Timeout:     "30s",
				Failures:    3,
			},
			expectError: true,
		},
		{
			name: "invalid timeout duration",
			config: goresilience.CircuitBreaker{
				MaxRequests: 5,
				Interval:    "10s",
				Timeout:     "invalid",
				Failures:    3,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := goresilience.Config{
				CircuitBreakers: map[string]goresilience.CircuitBreaker{
					"test_cb": tt.config,
				},
				Targets: map[string]goresilience.PolicyNames{
					"test_target": {
						CircuitBreaker: "test_cb",
					},
				},
			}

			_, err := goresilience.FromConfig(cfg)
			if tt.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestCircuitBreakerConcurrency(t *testing.T) {
	target := "test_target"
	cfg := goresilience.Config{
		CircuitBreakers: map[string]goresilience.CircuitBreaker{
			"test_cb": {
				MaxRequests: 1,
				Interval:    "1s",
				Timeout:     "500ms",
				Failures:    1,
			},
		},
		Targets: map[string]goresilience.PolicyNames{
			target: {
				CircuitBreaker: "test_cb",
			},
		},
	}

	provider, err := goresilience.FromConfig(cfg)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	policy := provider.Policy(target)
	exec := goresilience.NewExecWithPolicy(context.Background(), policy)

	// Run multiple goroutines to test concurrent access
	const numGoroutines = 10
	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			_, err := exec(func(ctx context.Context) (any, error) {
				// Simulate some work
				time.Sleep(10 * time.Millisecond)
				if id%2 == 0 {
					return successResult, nil
				}
				return nil, testError
			})
			results <- err
		}(i)
	}

	// Collect results
	var successCount, errorCount int
	for i := 0; i < numGoroutines; i++ {
		err := <-results
		if err == nil {
			successCount++
		} else {
			errorCount++
		}
	}

	// Verify we got some results (exact counts depend on timing and circuit state)
	if successCount+errorCount != numGoroutines {
		t.Fatalf("expected %d total results, got %d", numGoroutines, successCount+errorCount)
	}

	t.Logf("Concurrent test results: %d successes, %d errors", successCount, errorCount)
}
