package anker

import (
	"testing"
	"time"
)

func TestRateLimiterBasic(t *testing.T) {
	rl := NewRateLimiter()
	
	// Check defaults
	if rl.endpointLimit != DefaultEndpointLimit {
		t.Errorf("Expected endpoint limit %d, got %d", DefaultEndpointLimit, rl.endpointLimit)
	}
	
	// First request should not throttle
	throttle := rl.Wait("/test/endpoint")
	if throttle > 100*time.Millisecond {
		t.Errorf("First request should not be throttled significantly, got %v", throttle)
	}
}

func TestRateLimiterEndpointLimit(t *testing.T) {
	rl := NewRateLimiter()
	rl.SetEndpointLimit(3) // Set low limit for testing
	rl.SetRequestDelay(0)  // Disable delay for this test
	
	endpoint := "/test/endpoint"
	
	// Make 3 requests (at the limit)
	for i := 0; i < 3; i++ {
		throttle := rl.Wait(endpoint)
		if throttle > 10*time.Millisecond {
			t.Errorf("Request %d should not be throttled, got %v", i+1, throttle)
		}
		// Small sleep to distinguish timestamps
		time.Sleep(time.Millisecond)
	}
	
	// 4th request should be throttled
	start := time.Now()
	throttle := rl.Wait(endpoint)
	elapsed := time.Since(start)
	
	// It should have been throttled (waited some time)
	if throttle == 0 || elapsed < throttle-10*time.Millisecond {
		t.Errorf("4th request should be throttled significantly (got throttle=%v, elapsed=%v)", throttle, elapsed)
	}
	t.Logf("4th request throttled for %v (actual wait: %v)", throttle, elapsed)
}

func TestRateLimiterMultipleEndpoints(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping rate limiter test in short mode")
	}
	
	rl := NewRateLimiter()
	rl.SetEndpointLimit(2)
	rl.SetRequestDelay(0)
	
	endpoint1 := "/endpoint/1"
	endpoint2 := "/endpoint/2"
	
	// Test that each endpoint has its own independent limit
	// We'll test them sequentially to avoid the 60s sleep affecting both
	
	// Test endpoint 1
	rl.Wait(endpoint1)
	time.Sleep(2 * time.Millisecond)
	rl.Wait(endpoint1)
	time.Sleep(2 * time.Millisecond)
	
	// 3rd request to endpoint 1 should throttle
	start := time.Now()
	throttle1 := rl.Wait(endpoint1)
	elapsed := time.Since(start)
	
	if throttle1 == 0 {
		t.Error("Endpoint 1 should be throttled on 3rd request")
	}
	// Should throttle for approximately the time remaining in the minute window
	if elapsed < 50*time.Second {
		t.Errorf("Endpoint 1 throttle time too short: %v (elapsed: %v)", throttle1, elapsed)
	}
	t.Logf("Endpoint 1 throttled for %v (elapsed: %v)", throttle1, elapsed)
	
	// Test endpoint 2 independently (fresh requests after endpoint1 throttle)
	rl.Wait(endpoint2)
	time.Sleep(2 * time.Millisecond)
	rl.Wait(endpoint2)
	time.Sleep(2 * time.Millisecond)
	
	// 3rd request to endpoint 2 should also throttle
	start = time.Now()
	throttle2 := rl.Wait(endpoint2)
	elapsed = time.Since(start)
	
	if throttle2 == 0 {
		t.Error("Endpoint 2 should be throttled on 3rd request")
	}
	if elapsed < 50*time.Second {
		t.Errorf("Endpoint 2 throttle time too short: %v (elapsed: %v)", throttle2, elapsed)
	}
	t.Logf("Endpoint 2 throttled for %v (elapsed: %v)", throttle2, elapsed)
}

func TestRateLimiterRequestDelay(t *testing.T) {
	rl := NewRateLimiter()
	rl.SetEndpointLimit(0) // Disable endpoint limit
	rl.SetRequestDelay(100 * time.Millisecond)
	
	endpoint := "/test/endpoint"
	
	// First request
	start := time.Now()
	rl.Wait(endpoint)
	
	// Second request should be delayed
	rl.Wait(endpoint)
	elapsed := time.Since(start)
	
	if elapsed < 90*time.Millisecond {
		t.Errorf("Expected at least 90ms delay, got %v", elapsed)
	}
}

func TestRateLimiterStats(t *testing.T) {
	rl := NewRateLimiter()
	rl.SetRequestDelay(0)
	
	endpoint1 := "/endpoint/1"
	endpoint2 := "/endpoint/2"
	
	rl.Wait(endpoint1)
	rl.Wait(endpoint1)
	rl.Wait(endpoint2)
	
	stats := rl.GetStats()
	
	totalRequests := stats["total_requests_last_minute"].(int)
	if totalRequests != 3 {
		t.Errorf("Expected 3 total requests, got %d", totalRequests)
	}
	
	endpointCounts := stats["endpoint_counts"].(map[string]int)
	if endpointCounts[endpoint1] != 2 {
		t.Errorf("Expected 2 requests for endpoint1, got %d", endpointCounts[endpoint1])
	}
	if endpointCounts[endpoint2] != 1 {
		t.Errorf("Expected 1 request for endpoint2, got %d", endpointCounts[endpoint2])
	}
}

func TestRateLimiterCleanup(t *testing.T) {
	rl := NewRateLimiter()
	rl.SetRequestDelay(0)
	
	endpoint := "/test/endpoint"
	
	// Add some requests
	rl.Wait(endpoint)
	rl.Wait(endpoint)
	
	// Check we have 2 requests
	stats := rl.GetStats()
	if stats["total_requests_last_minute"].(int) != 2 {
		t.Error("Expected 2 requests")
	}
	
	// Manually trigger cleanup with time in the future
	rl.mu.Lock()
	rl.cleanOldRequests(time.Now().Add(2 * time.Minute))
	rl.mu.Unlock()
	
	// All requests should be cleaned
	stats = rl.GetStats()
	if stats["total_requests_last_minute"].(int) != 0 {
		t.Error("Expected 0 requests after cleanup")
	}
}
