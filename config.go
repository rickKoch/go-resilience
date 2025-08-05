package goresilience

type Config struct {
	Timeouts        map[string]string         `json:"timeouts,omitempty" yaml:"timeouts,omitempty"`
	Retries         map[string]Retry          `json:"retries,omitempty" yaml:"retries,omitempty"`
	CircuitBreakers map[string]CircuitBreaker `json:"circuitBreakers,omitempty" yaml:"circuitBreakers,omitempty"`
	Targets         map[string]PolicyNames    `json:"targets,omitempty" yaml:"targets,omitempty"`
}

type Retry struct {
	Duration   string `json:"duration,omitempty" yaml:"duration,omitempty"`
	MaxRetries int    `json:"maxRetries,omitempty" yaml:"maxRetries,omitempty"`
}

type CircuitBreaker struct {
	MaxRequests int    `json:"maxRequests,omitempty" yaml:"maxRequests,omitempty"`
	Interval    string `json:"interval,omitempty" yaml:"interval,omitempty"`
	Timeout     string `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Failures    int    `json:"failures,omitempty" yaml:"failures,omitempty"`
}

type PolicyNames struct {
	Timeout        string `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Retry          string `json:"retry,omitempty" yaml:"retry,omitempty"`
	CircuitBreaker string `json:"circuitBreaker,omitempty" yaml:"circuitBreaker,omitempty"`
}
