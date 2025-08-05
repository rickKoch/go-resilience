package goresilience

import (
	"fmt"
	"strconv"
	"time"
)

type target struct {
	timeout        string
	retry          string
	circuitBreaker string
}

type Provider struct {
	timeouts        map[string]time.Duration
	retries         map[string]*retry
	circuitBreakers map[string]*circuitBreaker
	targets         map[string]target
}

func FromConfig(cfg Config) (*Provider, error) {
	p := &Provider{
		timeouts:        make(map[string]time.Duration),
		retries:         make(map[string]*retry),
		circuitBreakers: make(map[string]*circuitBreaker),
		targets:         make(map[string]target),
	}

	if err := p.configure(cfg); err != nil {
		return nil, err
	}

	return p, nil
}

func (p *Provider) Policy(target string) *Policy {
	policy := &Policy{}

	if cfg, ok := p.targets[target]; ok {
		if cfg.timeout != "" {
			if timeout, exists := p.timeouts[cfg.timeout]; exists {
				policy.timeout = timeout
			}
		}

		if cfg.retry != "" {
			if retry, exists := p.retries[cfg.retry]; exists {
				policy.retry = retry
			}
		}

		if cfg.circuitBreaker != "" {
			if cb, exists := p.circuitBreakers[cfg.circuitBreaker]; exists {
				policy.circuitBreaker = cb
			}
		}
	}

	return policy
}

func (p *Provider) configure(cfg Config) error {
	for name, val := range cfg.Timeouts {
		timeout, err := parseDuration(val)
		if err != nil {
			return fmt.Errorf("invalid timeout duration %s for %q: %w", val, name, err)
		}
		p.timeouts[name] = timeout
	}

	for name, retryCfg := range cfg.Retries {
		retryInstance, err := newRetry(name, retryCfg)
		if err != nil {
			return fmt.Errorf("failed to create retry for %q: %w", name, err)
		}

		p.retries[name] = retryInstance
	}

	for name, cbCfg := range cfg.CircuitBreakers {
		cb, err := newCircuitBreaker(name, cbCfg)
		if err != nil {
			return fmt.Errorf("failed to create circuit breaker for %q: %w", name, err)
		}

		p.circuitBreakers[name] = cb
	}

	for k, n := range cfg.Targets {
		p.targets[k] = target{
			timeout:        n.Timeout,
			retry:          n.Retry,
			circuitBreaker: n.CircuitBreaker,
		}
	}
	return nil
}

func parseDuration(val string) (time.Duration, error) {
	if val == "" {
		return 0, nil
	}

	if i, err := strconv.ParseInt(val, 10, 64); err == nil {
		return time.Duration(i) * time.Microsecond, nil
	}

	return time.ParseDuration(val)
}
