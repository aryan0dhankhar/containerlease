package circuitbreaker

import (
	"sync"
	"sync/atomic"
	"time"
)

// State represents the circuit breaker state
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

// CircuitBreaker provides fast-fail behavior when a dependency fails repeatedly
type CircuitBreaker struct {
	state            atomic.Value
	failureCount     atomic.Int32
	successCount     atomic.Int32
	lastFailureTime  atomic.Value
	failureThreshold int32
	successThreshold int32
	timeout          time.Duration
	mu               sync.RWMutex
	onStateChange    func(from, to State)
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(failureThreshold, successThreshold int32, timeout time.Duration) *CircuitBreaker {
	cb := &CircuitBreaker{
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		timeout:          timeout,
		onStateChange:    func(_, _ State) {},
	}
	cb.state.Store(StateClosed)
	return cb
}

// SetStateChangeCallback registers a callback for state transitions
func (cb *CircuitBreaker) SetStateChangeCallback(fn func(from, to State)) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.onStateChange = fn
}

// RecordSuccess increments success counter and attempts half-open -> closed transition
func (cb *CircuitBreaker) RecordSuccess() {
	currentState := cb.GetState()
	switch currentState {
	case StateHalfOpen:
		cb.successCount.Add(1)
		if cb.successCount.Load() >= cb.successThreshold {
			cb.setState(StateClosed)
			cb.failureCount.Store(0)
			cb.successCount.Store(0)
		}
	case StateClosed:
		cb.failureCount.Store(0)
	}
}

// RecordFailure increments failure counter and may trip open or stay open
func (cb *CircuitBreaker) RecordFailure() {
	currentState := cb.GetState()
	now := time.Now()
	cb.lastFailureTime.Store(&now)

	switch currentState {
	case StateClosed:
		cb.failureCount.Add(1)
		if cb.failureCount.Load() >= cb.failureThreshold {
			cb.setState(StateOpen)
			cb.failureCount.Store(0)
			cb.successCount.Store(0)
		}
	case StateHalfOpen:
		cb.setState(StateOpen)
		cb.failureCount.Store(0)
		cb.successCount.Store(0)
	}
}

// AllowRequest returns true if the circuit allows a request
func (cb *CircuitBreaker) AllowRequest() bool {
	currentState := cb.GetState()
	if currentState == StateClosed {
		return true
	}
	if currentState == StateHalfOpen {
		return true
	}
	lastFailure, ok := cb.lastFailureTime.Load().(*time.Time)
	if !ok || lastFailure == nil {
		return false
	}
	if time.Since(*lastFailure) > cb.timeout {
		cb.setState(StateHalfOpen)
		cb.failureCount.Store(0)
		cb.successCount.Store(0)
		return true
	}
	return false
}

// GetState returns the current state
func (cb *CircuitBreaker) GetState() State {
	return cb.state.Load().(State)
}

// setState transitions to a new state and calls the callback
func (cb *CircuitBreaker) setState(newState State) {
	oldState := cb.GetState()
	if oldState == newState {
		return
	}
	cb.state.Store(newState)
	cb.mu.RLock()
	fn := cb.onStateChange
	cb.mu.RUnlock()
	if fn != nil {
		fn(oldState, newState)
	}
}
