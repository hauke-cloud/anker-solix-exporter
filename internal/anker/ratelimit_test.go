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
	rl := NewRateLimiter()
	rl.SetEndpointLimit(2)
	rl.SetRequestDelay(0)
	
	endpoint1 := "/endpoint/1"
	endpoint2 := "/endpoint/2"
	
	// Each endpoint should have its own limit
	rl.Wait(endpoint1)
	rl.Wait(endpoint1)
	rl.Wait(endpoint2)
	rl.Wait(endpoint2)
	
	// Both should throttle on 3rd request
	throttle1 := rl.Wait(endpoint1)
	throttle2 := rl.Wait(endpoint2)
	
	if throttle1 == 0 {
		t.Error("Endpoint 1 should be throttled")
	}
	if throttle2 == 0 {
		t.Error("Endpoint 2 should be throttled")
	}
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
