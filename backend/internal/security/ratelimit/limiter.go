package ratelimit

import (
	"sync"
	"time"
)

type Limiter struct {
	mu      sync.RWMutex
	buckets map[string]*bucket
	maxReqs int
	window  time.Duration
	cleanup *time.Ticker
}

type bucket struct {
	requests []time.Time
	lastSeen time.Time
}

func NewLimiter(maxRequests int, window time.Duration) *Limiter {
	limiter := &Limiter{
		buckets: make(map[string]*bucket),
		maxReqs: maxRequests,
		window:  window,
		cleanup: time.NewTicker(5 * time.Minute),
	}
	go limiter.cleanupOldBuckets()
	return limiter
}

func (l *Limiter) Allow(tenantID string) bool {
	if tenantID == "" {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	b, exists := l.buckets[tenantID]
	if !exists {
		b = &bucket{requests: []time.Time{}}
		l.buckets[tenantID] = b
	}

	cutoff := now.Add(-l.window)
	var reqs []time.Time
	for _, t := range b.requests {
		if t.After(cutoff) {
			reqs = append(reqs, t)
		}
	}
	b.requests = reqs
	b.lastSeen = now

	if len(b.requests) >= l.maxReqs {
		return false
	}

	b.requests = append(b.requests, now)
	return true
}

// AllowStrict allows requests with stricter limits for sensitive endpoints
func (l *Limiter) AllowStrict(identifier string, maxReqs int, window time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Use a separate key for strict limits to avoid conflicts
	key := "strict:" + identifier
	now := time.Now()
	b, exists := l.buckets[key]
	if !exists {
		b = &bucket{requests: []time.Time{}}
		l.buckets[key] = b
	}

	cutoff := now.Add(-window)
	var reqs []time.Time
	for _, t := range b.requests {
		if t.After(cutoff) {
			reqs = append(reqs, t)
		}
	}
	b.requests = reqs
	b.lastSeen = now

	if len(b.requests) >= maxReqs {
		return false
	}

	b.requests = append(b.requests, now)
	return true
}

func (l *Limiter) cleanupOldBuckets() {
	for range l.cleanup.C {
		l.mu.Lock()
		now := time.Now()
		staleThreshold := now.Add(-15 * time.Minute)
		for tenantID, b := range l.buckets {
			if b.lastSeen.Before(staleThreshold) {
				delete(l.buckets, tenantID)
			}
		}
		l.mu.Unlock()
	}
}

func (l *Limiter) Stop() {
	l.cleanup.Stop()
}
