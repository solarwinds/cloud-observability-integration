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
	"os"
	"strconv"
	"strings"
	"time"
)

// Environment variable names for VPC Flow Log configuration
const (
	VpcLogGroupNameVar    = "VPC_LOG_GROUP_NAME"             // CloudWatch log group name for VPC Flow Logs
	LogLevelVar           = "LOG_LEVEL"                      // Log level (DEBUG enables verbose logging)
	VpcDebugIntervalVar   = "VPC_DEBUG_INTERVAL"             // How often to log full JSON (every Nth record)
	VpcFlowLogCacheTTLVar = "VPC_FLOW_LOG_CACHE_TTL_MINUTES" // Cache TTL for flow log format in minutes
)

// Default configuration values
const (
	DefaultVpcFlowLogCacheTTLMinutes = 10  // Default cache TTL: 10 minutes
	DefaultVpcDebugInterval          = 100 // Default: log full JSON every 100th record
)

// Config holds VPC Flow Logs configuration and handler instance
type Config struct {
	LogGroupName string   // CloudWatch log group name to monitor for VPC Flow Logs
	Handler      *Handler // Handler instance for processing VPC Flow Logs
}

// InitializeFromEnv reads environment variables and creates a configured VPC Flow Logs handler
// This function encapsulates all VPC-specific initialization logic.
//
// Environment variables read:
//   - VPC_LOG_GROUP_NAME: CloudWatch log group name for VPC Flow Logs
//   - LOG_LEVEL: Set to "DEBUG" to enable verbose logging
//   - VPC_DEBUG_INTERVAL: How often to log full JSON (default: 100)
//   - VPC_FLOW_LOG_CACHE_TTL_MINUTES: Cache TTL in minutes (default: 10)
//
// Returns:
//   - Config with log group name and initialized handler
//   - Handler will be nil if VPC_LOG_GROUP_NAME is not set (VPC processing disabled)
func InitializeFromEnv() *Config {
	logGroupName := os.Getenv(VpcLogGroupNameVar)

	// If no VPC log group is configured, return config with nil handler
	// This indicates VPC Flow Log processing is disabled
	if logGroupName == "" {
		return &Config{
			LogGroupName: "",
			Handler:      nil,
		}
	}

	// Parse configuration from environment variables
	isDebugEnabled := strings.EqualFold(os.Getenv(LogLevelVar), "DEBUG")
	debugInterval := parseVpcDebugInterval()
	cacheTTL := parseVpcFlowLogCacheTTL()

	// Create the handler with parsed configuration
	handler := NewHandler(isDebugEnabled, debugInterval, cacheTTL)

	if isDebugEnabled {
		handlerLogger.Info(fmt.Sprintf("VPC handler initialized with cache TTL: %v", cacheTTL))
	}

	return &Config{
		LogGroupName: logGroupName,
		Handler:      handler,
	}
}

// IsEnabled returns true if VPC Flow Log processing is enabled (handler is configured)
func (c *Config) IsEnabled() bool {
	return c.Handler != nil && c.LogGroupName != ""
}

// ShouldProcess returns true if the given log group should be processed as VPC Flow Logs
func (c *Config) ShouldProcess(logGroup string) bool {
	return c.IsEnabled() && logGroup == c.LogGroupName
}

// parseVpcFlowLogCacheTTL parses the VPC_FLOW_LOG_CACHE_TTL_MINUTES environment variable
// Returns a safe default of 10 minutes if not set or invalid
func parseVpcFlowLogCacheTTL() time.Duration {
	cacheTTLStr := os.Getenv(VpcFlowLogCacheTTLVar)
	if cacheTTLStr == "" {
		return DefaultVpcFlowLogCacheTTLMinutes * time.Minute
	}

	cacheTTLMinutes, err := strconv.Atoi(cacheTTLStr)
	if err != nil {
		handlerLogger.Error(fmt.Sprintf("VPC_FLOW_LOG_CACHE_TTL_MINUTES: unable to parse '%s' as number, using default %d minutes",
			cacheTTLStr, DefaultVpcFlowLogCacheTTLMinutes))
		return DefaultVpcFlowLogCacheTTLMinutes * time.Minute
	}

	return time.Duration(cacheTTLMinutes) * time.Minute
}

// parseVpcDebugInterval parses the VPC_DEBUG_INTERVAL environment variable
// Returns a safe default of 100 if not set or invalid
func parseVpcDebugInterval() int {
	intervalStr := os.Getenv(VpcDebugIntervalVar)
	if intervalStr == "" {
		return DefaultVpcDebugInterval
	}

	interval, err := strconv.Atoi(intervalStr)
	if err != nil {
		handlerLogger.Error(fmt.Sprintf("VPC_DEBUG_INTERVAL: unable to parse '%s' as number, using default %d",
			intervalStr, DefaultVpcDebugInterval))
		return DefaultVpcDebugInterval
	}

	// Check boundary conditions with specific error messages
	if interval < 1 {
		handlerLogger.Error(fmt.Sprintf("VPC_DEBUG_INTERVAL can't be less than 1, got %d, using default %d",
			interval, DefaultVpcDebugInterval))
		return DefaultVpcDebugInterval
	}

	// Set reasonable upper bounds
	if interval > 10000 {
		handlerLogger.Error(fmt.Sprintf("VPC_DEBUG_INTERVAL too large (max 10000), got %d, capping at 10000", interval))
		return 10000
	}

	return interval
}
