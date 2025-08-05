package goresilience

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
)

type retry struct {
	duration   time.Duration
	maxRetries int
}

func newRetry(name string, r Retry) (*retry, error) {
	duration, err := parseDuration(r.Duration)
	if err != nil {
		return nil, fmt.Errorf("invalid retry duration %s for '%q': %w", r.Duration, name, err)
	}

	return &retry{duration, r.MaxRetries}, nil
}

func (r *retry) backoff(ctx context.Context) backoff.BackOff {
	var b backoff.BackOff = backoff.NewConstantBackOff(r.duration)

	if r.maxRetries >= 0 {
		b = backoff.WithMaxRetries(b, uint64(r.maxRetries))
	}

	return backoff.WithContext(b, ctx)
}

func OperationRetry(operation backoff.OperationWithData[any], b backoff.BackOff) (any, error) {
	return backoff.RetryWithData(func() (any, error) {
		return operation()
	}, b)
}
