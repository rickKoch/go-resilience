package goresilience

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
)

type Operation func(ctx context.Context) (any, error)

type Executor func(oper Operation) (any, error)

type operationResult struct {
	value any
	err   error
}

type Policy struct {
	timeout        time.Duration
	retry          *retry
	circuitBreaker *circuitBreaker
}

func NewExecutor(ctx context.Context, policy *Policy) Executor {
	if policy == nil {
		return func(oper Operation) (any, error) {
			return oper(ctx)
		}
	}

	return func(oper Operation) (any, error) {
		operation := oper

		if policy.timeout > 0 {
			operation = policy.withTimeout(operation)
		}

		if policy.circuitBreaker != nil {
			operation = policy.withCircuitBreaker(operation)
		}

		if policy.retry == nil {
			return operation(ctx)
		}

		return policy.withRetry(ctx, operation)
	}
}

func NewExecWithPolicy(ctx context.Context, policy *Policy) Executor {
	return NewExecutor(ctx, policy)
}

func (p *Policy) withTimeout(oper Operation) Operation {
	return func(ctx context.Context) (any, error) {
		timeoutCtx, cancel := context.WithTimeout(ctx, p.timeout)
		defer cancel()

		resultCh := make(chan operationResult, 1)

		go func() {
			defer func() {
				if r := recover(); r != nil {
					// Handle panics gracefully
					select {
					case resultCh <- operationResult{nil, fmt.Errorf("operation panicked: %v", r)}:
					default:
					}
				}
			}()

			value, err := oper(timeoutCtx)

			select {
			case resultCh <- operationResult{value, err}:
			case <-timeoutCtx.Done():
				// Operation completed but context already timed out
			}
		}()

		// Wait for either operation completion or timeout
		select {
		case result := <-resultCh:
			return result.value, result.err
		case <-timeoutCtx.Done():
			return nil, timeoutCtx.Err()
		}
	}
}

func (p *Policy) withCircuitBreaker(oper Operation) Operation {
	return func(ctx context.Context) (any, error) {
		res, err := p.circuitBreaker.breaker.Execute(func() (any, error) {
			return oper(ctx)
		})

		if p.retry != nil && IsErrorPermanent(err) {
			err = backoff.Permanent(err)
		}

		return res, err
	}
}

func (p *Policy) withRetry(ctx context.Context, oper Operation) (any, error) {
	return OperationRetry(func() (any, error) {
		return oper(ctx)
	}, p.retry.backoff(ctx))
}
