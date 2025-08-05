package goresilience

import (
	"errors"

	"github.com/sony/gobreaker"
)

var (
	ErrOpenState       = gobreaker.ErrOpenState
	ErrTooManyRequests = gobreaker.ErrTooManyRequests
)

type circuitBreaker struct {
	breaker *gobreaker.CircuitBreaker
}

func newCircuitBreaker(name string, config CircuitBreaker) (*circuitBreaker, error) {
	interval, err := parseDuration(config.Interval)
	if err != nil {
		return nil, err
	}

	timeout, err := parseDuration(config.Timeout)
	if err != nil {
		return nil, err
	}
	maxRequest := uint32(config.MaxRequests)
	failures := uint32(config.Failures)

	cb := new(circuitBreaker)

	tripFn := func(counts gobreaker.Counts) bool {
		return counts.ConsecutiveFailures >= failures
	}

	cb.breaker = gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        name,
		MaxRequests: maxRequest,
		Interval:    interval,
		Timeout:     timeout,
		ReadyToTrip: tripFn,
	})

	return cb, nil
}

func (cb *circuitBreaker) State() gobreaker.State {
	return cb.breaker.State()
}

func (cb *circuitBreaker) Counts() gobreaker.Counts {
	return cb.breaker.Counts()
}

func IsErrorPermanent(err error) bool {
	return errors.Is(err, ErrOpenState) || errors.Is(err, ErrTooManyRequests)
}
