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
	"fmt"
	"sync"
	"time"
)

// flowLogFormatCacheEntry represents a cached flow log format with metadata
type flowLogFormatCacheEntry struct {
	logFormat     string
	flowLogId     string
	flowLogsCount int
	cachedAt      time.Time
}

// flowLogFormatCache manages caching of flow log formats to reduce EC2 API calls
type flowLogFormatCache struct {
	mu             sync.RWMutex
	entries        map[string]*flowLogFormatCacheEntry
	cacheTTL       time.Duration
	isDebugEnabled bool
}

// newFlowLogFormatCache creates a new cache with the specified TTL
func newFlowLogFormatCache(cacheTTL time.Duration, isDebugEnabled bool) *flowLogFormatCache {
	if isDebugEnabled {
		handlerLogger.Info(fmt.Sprintf("Initializing in-memory flow log format cache with TTL: %v", cacheTTL))
	}
	return &flowLogFormatCache{
		entries:        make(map[string]*flowLogFormatCacheEntry),
		cacheTTL:       cacheTTL,
		isDebugEnabled: isDebugEnabled,
	}
}

// get retrieves a cached entry if it exists and hasn't expired
func (c *flowLogFormatCache) get(logGroupName string) (string, string, int, bool) {
	c.mu.RLock()

	entry, exists := c.entries[logGroupName]
	if !exists {
		c.mu.RUnlock()
		if c.isDebugEnabled {
			handlerLogger.Info(fmt.Sprintf("✗ Cache MISS for log group: %s", logGroupName))
		}
		return "", "", 0, false
	}

	// Check if cache entry has expired
	if time.Since(entry.cachedAt) > c.cacheTTL {
		if c.isDebugEnabled {
			handlerLogger.Info(fmt.Sprintf("✗ Cache EXPIRED for log group: %s (age: %v, TTL: %v)", logGroupName, time.Since(entry.cachedAt), c.cacheTTL))
		}

		// Delete expired entry to prevent memory leak
		// Upgrade from read lock to write lock
		c.mu.RUnlock()
		c.mu.Lock()
		// Double-check: entry might have been updated by another goroutine
		if entry, exists := c.entries[logGroupName]; exists && time.Since(entry.cachedAt) > c.cacheTTL {
			delete(c.entries, logGroupName)
			if c.isDebugEnabled {
				handlerLogger.Info(fmt.Sprintf("✗ Cache EXPIRED entry DELETED for log group: %s", logGroupName))
			}
		}
		c.mu.Unlock()
		return "", "", 0, false
	}

	if c.isDebugEnabled {
		handlerLogger.Info(fmt.Sprintf("✓ Cache HIT for log group: %s | Format: %s | FlowLogId: %s (age: %v)", logGroupName, entry.logFormat, entry.flowLogId, time.Since(entry.cachedAt)))
	}

	c.mu.RUnlock()
	return entry.logFormat, entry.flowLogId, entry.flowLogsCount, true
}

// set stores a new cache entry
func (c *flowLogFormatCache) set(logGroupName, logFormat, flowLogId string, flowLogsCount int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[logGroupName] = &flowLogFormatCacheEntry{
		logFormat:     logFormat,
		flowLogId:     flowLogId,
		flowLogsCount: flowLogsCount,
		cachedAt:      time.Now(),
	}

	if c.isDebugEnabled {
		handlerLogger.Info(fmt.Sprintf("✓ Cached format for log group: %s | Format: %s | FlowLogId: %s", logGroupName, logFormat, flowLogId))
	}
}
