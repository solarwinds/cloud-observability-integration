/* Copyright 2022 SolarWinds Worldwide, LLC. All rights reserved.
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at:
*
*	http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and limitations
* under the License.
 */

package vpc_flow_logs

import (
	"testing"
	"time"
)

func TestFlowLogFormatCache_CacheMiss(t *testing.T) {
	cache := newFlowLogFormatCache(10*time.Minute, false)

	logFormat, flowLogId, flowLogsCount, found := cache.get("test-log-group")

	if found {
		t.Error("Expected cache miss, but got a hit")
	}
	if logFormat != "" || flowLogId != "" || flowLogsCount != 0 {
		t.Errorf("Expected empty values on cache miss, got: logFormat=%s, flowLogId=%s, flowLogsCount=%d",
			logFormat, flowLogId, flowLogsCount)
	}
}

func TestFlowLogFormatCache_CacheHit(t *testing.T) {
	cache := newFlowLogFormatCache(10*time.Minute, false)

	// Set a value
	cache.set("test-log-group", "${version} ${account-id}", "fl-12345", 5)

	// Get it back
	logFormat, flowLogId, flowLogsCount, found := cache.get("test-log-group")

	if !found {
		t.Error("Expected cache hit, but got a miss")
	}
	if logFormat != "${version} ${account-id}" {
		t.Errorf("Expected logFormat '${version} ${account-id}', got '%s'", logFormat)
	}
	if flowLogId != "fl-12345" {
		t.Errorf("Expected flowLogId 'fl-12345', got '%s'", flowLogId)
	}
	if flowLogsCount != 5 {
		t.Errorf("Expected flowLogsCount 5, got %d", flowLogsCount)
	}
}

func TestFlowLogFormatCache_CacheExpiry(t *testing.T) {
	// Create cache with very short TTL (100ms)
	cache := newFlowLogFormatCache(100*time.Millisecond, false)

	// Set a value
	cache.set("test-log-group", "${version} ${account-id}", "fl-12345", 5)

	// Verify it's cached
	_, _, _, found := cache.get("test-log-group")
	if !found {
		t.Error("Expected cache hit immediately after set")
	}

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// This should trigger the expiry path and delete the entry
	// This is the critical test for the double-RUnlock bug fix
	logFormat, flowLogId, flowLogsCount, found := cache.get("test-log-group")

	if found {
		t.Error("Expected cache miss after expiry, but got a hit")
	}
	if logFormat != "" || flowLogId != "" || flowLogsCount != 0 {
		t.Errorf("Expected empty values on expired cache entry, got: logFormat=%s, flowLogId=%s, flowLogsCount=%d",
			logFormat, flowLogId, flowLogsCount)
	}

	// Verify the entry was actually deleted from the cache
	cache.mu.RLock()
	_, exists := cache.entries["test-log-group"]
	cache.mu.RUnlock()

	if exists {
		t.Error("Expected expired entry to be deleted from cache, but it still exists")
	}
}

func TestFlowLogFormatCache_ConcurrentAccess(t *testing.T) {
	cache := newFlowLogFormatCache(10*time.Minute, false)

	// Set initial value
	cache.set("test-log-group", "${version} ${account-id}", "fl-12345", 5)

	// Concurrently read from cache
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				cache.get("test-log-group")
			}
			done <- true
		}()
	}

	// Wait for all goroutines to finish
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify cache still has the value
	logFormat, flowLogId, flowLogsCount, found := cache.get("test-log-group")
	if !found {
		t.Error("Expected cache hit after concurrent reads")
	}
	if logFormat != "${version} ${account-id}" || flowLogId != "fl-12345" || flowLogsCount != 5 {
		t.Error("Cache value was corrupted during concurrent access")
	}
}

func TestFlowLogFormatCache_ConcurrentExpiryAccess(t *testing.T) {
	// This test specifically exercises the expiry path with concurrent access
	// to ensure the double-RUnlock bug is fixed
	cache := newFlowLogFormatCache(50*time.Millisecond, false)

	// Set initial value
	cache.set("test-log-group", "${version} ${account-id}", "fl-12345", 5)

	// Wait for expiry
	time.Sleep(60 * time.Millisecond)

	// Concurrently try to get the expired entry
	// This should not panic with "sync: RUnlock of unlocked RWMutex"
	done := make(chan bool)
	for i := 0; i < 20; i++ {
		go func() {
			cache.get("test-log-group")
			done <- true
		}()
	}

	// Wait for all goroutines to finish
	// If there's a double-RUnlock bug, this will panic
	for i := 0; i < 20; i++ {
		<-done
	}

	// If we reach here, the fix is working correctly
	t.Log("Successfully handled concurrent access to expired cache entry without panic")
}

func TestFlowLogFormatCache_SetUpdatesExistingEntry(t *testing.T) {
	cache := newFlowLogFormatCache(10*time.Minute, false)

	// Set initial value
	cache.set("test-log-group", "${version} ${account-id}", "fl-12345", 5)

	// Update with new value
	cache.set("test-log-group", "${version} ${srcaddr} ${dstaddr}", "fl-67890", 10)

	// Get the updated value
	logFormat, flowLogId, flowLogsCount, found := cache.get("test-log-group")

	if !found {
		t.Error("Expected cache hit after update")
	}
	if logFormat != "${version} ${srcaddr} ${dstaddr}" {
		t.Errorf("Expected updated logFormat, got '%s'", logFormat)
	}
	if flowLogId != "fl-67890" {
		t.Errorf("Expected updated flowLogId 'fl-67890', got '%s'", flowLogId)
	}
	if flowLogsCount != 10 {
		t.Errorf("Expected updated flowLogsCount 10, got %d", flowLogsCount)
	}
}
