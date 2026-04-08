package anker

import (
	"sync"
	"time"
)

// Rate limiting configuration based on Anker API limits
const (
	// Maximum number of requests per endpoint per minute
	DefaultEndpointLimit = 10
	
	// Minimum delay between requests (in seconds)
	DefaultRequestDelay = 0.3
	
	// Request timeout (in seconds)
	DefaultRequestTimeout = 10
)

// endpointRequest tracks a single request to an endpoint
type endpointRequest struct {
	endpoint  string
	timestamp time.Time
}

// RateLimiter tracks API request rates per endpoint
type RateLimiter struct {
	mu                sync.RWMutex
	endpointLimit     int                    // Max requests per endpoint per minute
	requestDelay      time.Duration          // Minimum delay between requests
	requests          []endpointRequest      // Request history
	lastRequestTime   time.Time              // Last request timestamp
	endpointCounters  map[string]int         // Current count per endpoint in sliding window
}

// NewRateLimiter creates a new rate limiter with default settings
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		endpointLimit:    DefaultEndpointLimit,
		requestDelay:     time.Duration(DefaultRequestDelay * float64(time.Second)),
		requests:         make([]endpointRequest, 0),
		endpointCounters: make(map[string]int),
	}
}

// SetEndpointLimit sets the maximum requests per endpoint per minute
func (rl *RateLimiter) SetEndpointLimit(limit int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	if limit < 0 {
		limit = 0
	}
	rl.endpointLimit = limit
}

// SetRequestDelay sets the minimum delay between requests
func (rl *RateLimiter) SetRequestDelay(delay time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	rl.requestDelay = delay
}

// Wait blocks until it's safe to make a request to the given endpoint
// Returns the throttle duration (0 if no throttling needed)
func (rl *RateLimiter) Wait(endpoint string) time.Duration {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	throttle := time.Duration(0)

	// Clean old requests (older than 1 minute)
	rl.cleanOldRequests(now)

	// Check endpoint-specific rate limit
	if rl.endpointLimit > 0 {
		sameRequests := rl.countEndpointRequests(endpoint, now)
		
		if sameRequests >= rl.endpointLimit {
			// Calculate throttle time until oldest request expires
			oldestTime := rl.getOldestRequestTime(endpoint, now)
			throttle = time.Until(oldestTime.Add(time.Minute))
			if throttle < 0 {
				throttle = 0
			}
		}
	}

	// Apply minimum delay between any requests
	if !rl.lastRequestTime.IsZero() {
		timeSinceLastRequest := now.Sub(rl.lastRequestTime)
		delayNeeded := rl.requestDelay - timeSinceLastRequest
		
		if delayNeeded > throttle {
			throttle = delayNeeded
		}
	}

	// Wait if throttling is needed
	if throttle > 0 {
		time.Sleep(throttle)
	}

	// Record this request
	rl.lastRequestTime = time.Now()
	rl.requests = append(rl.requests, endpointRequest{
		endpoint:  endpoint,
		timestamp: rl.lastRequestTime,
	})

	return throttle
}

// cleanOldRequests removes requests older than 1 minute
func (rl *RateLimiter) cleanOldRequests(now time.Time) {
	cutoff := now.Add(-time.Minute)
	newRequests := make([]endpointRequest, 0, len(rl.requests))
	
	for _, req := range rl.requests {
		if req.timestamp.After(cutoff) {
			newRequests = append(newRequests, req)
		}
	}
	
	rl.requests = newRequests
}

// countEndpointRequests counts requests to a specific endpoint in the last minute
func (rl *RateLimiter) countEndpointRequests(endpoint string, now time.Time) int {
	cutoff := now.Add(-time.Minute)
	count := 0
	
	for _, req := range rl.requests {
		if req.endpoint == endpoint && req.timestamp.After(cutoff) {
			count++
		}
	}
	
	return count
}

// getOldestRequestTime finds the oldest request timestamp for an endpoint
func (rl *RateLimiter) getOldestRequestTime(endpoint string, now time.Time) time.Time {
	cutoff := now.Add(-time.Minute)
	var oldest time.Time
	
	for _, req := range rl.requests {
		if req.endpoint == endpoint && req.timestamp.After(cutoff) {
			if oldest.IsZero() || req.timestamp.Before(oldest) {
				oldest = req.timestamp
			}
		}
	}
	
	return oldest
}

// GetStats returns statistics about current rate limiting
func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	now := time.Now()
	cutoff := now.Add(-time.Minute)
	
	stats := make(map[string]interface{})
	endpointCounts := make(map[string]int)
	totalRequests := 0
	
	for _, req := range rl.requests {
		if req.timestamp.After(cutoff) {
			totalRequests++
			endpointCounts[req.endpoint]++
		}
	}
	
	stats["total_requests_last_minute"] = totalRequests
	stats["endpoint_counts"] = endpointCounts
	stats["endpoint_limit"] = rl.endpointLimit
	stats["request_delay"] = rl.requestDelay.Seconds()
	
	return stats
}
