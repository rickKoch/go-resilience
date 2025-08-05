# Go Resiliency

A lightweight Go library implementing resilience patterns for fault-tolerant distributed systems.

## Features

- **Timeout Pattern**: Prevent operations from running indefinitely
- **Retry Pattern**: Automatic retry with configurable backoff strategies  
- **Circuit Breaker Pattern**: Fail-fast mechanism for unhealthy services
- **Composable Policies**: Combine multiple resilience patterns
- **Context-aware**: Full support for Go's context package

## Installation

```bash
go get github.com/rickKoch/go-resilience
```

## Quick Start

```go
package main

import (
    "context"
    "time"
    
    goresilience "github.com/rickKoch/go-resilience"
)

func main() {
    ctx := context.Background()
    
    // Create a policy with timeout
    policy := &goresilience.Policy{}
    
    // Create executor
    executor := goresilience.NewExecutor(ctx, policy)
    
    // Execute operation with resilience
    result, err := executor(func(ctx context.Context) (any, error) {
        // Your operation here
        return callExternalService(ctx)
    })
    
    if err != nil {
        // Handle error
        return
    }
    
    // Use result
    _ = result
}
```

## Configuration-Based Usage

```go
// Define configuration
cfg := goresilience.Config{
    Timeouts: map[string]string{
        "fast": "100ms",
        "slow": "5s",
    },
    Retries: map[string]goresilience.RetryConfig{
        "gentle": {
            MaxAttempts: 3,
            BackoffType: "exponential",
            InitialDelay: "100ms",
        },
    },
    Targets: map[string]goresilience.TargetConfig{
        "user-service": {
            Timeout: "fast",
            Retry: "gentle",
        },
    },
}

// Create provider
provider, err := goresilience.FromConfig(cfg)
if err != nil {
    panic(err)
}

// Get policy for specific target
policy := provider.Policy("user-service")
executor := goresilience.NewExecutor(ctx, policy)
```

## Patterns

### Timeout
Automatically cancels operations that exceed specified duration:
```go
policy := &goresilience.Policy{
    Timeout: 5 * time.Second,
}
```

### Retry  
Retries failed operations with backoff strategies:
```go
retryConfig := goresilience.RetryConfig{
    MaxAttempts: 3,
    BackoffType: "exponential", // exponential, linear, constant
    InitialDelay: "100ms",
}
```

### Circuit Breaker
Prevents cascading failures by failing fast:
```go
cbConfig := goresilience.CircuitBreakerConfig{
    MaxRequests: 10,
    Interval: "60s", 
    Timeout: "30s",
}
```

## Pattern Composition

Patterns are applied in order: **Timeout → Circuit Breaker → Retry**

```go
// All patterns combined
policy := provider.Policy("critical-service")
// This policy applies timeout, circuit breaker, and retry as configured
```

## Dependencies

- [`github.com/cenkalti/backoff/v4`](https://github.com/cenkalti/backoff) - Retry backoff strategies
- [`github.com/sony/gobreaker`](https://github.com/sony/gobreaker) - Circuit breaker implementation

