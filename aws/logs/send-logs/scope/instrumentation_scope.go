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

package scope

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
)

const (
	// Telemetry scope
	ScopeName    = "vpc_flow_logs"
	ScopeVersion = "1.0.0"
	Identifier   = "nio"
	SwiReporter  = ""
)

// SetInstrumentationScope sets the instrumentation scope name, version, and attributes
// This is used for both logs and metrics to ensure consistency
func SetInstrumentationScope(scope pcommon.InstrumentationScope) {
	scope.SetName(ScopeName)
	scope.SetVersion(ScopeVersion)
	scope.Attributes().PutStr("identifier", Identifier)
	scope.Attributes().PutStr("swi-reporter", SwiReporter)
}
