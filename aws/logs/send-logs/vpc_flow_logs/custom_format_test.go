package vpc_flow_logs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCustomFormat_PartialFields tests parsing custom formats
// Custom formats MUST include all V2 (default format) fields to be accepted
func TestCustomFormat_PartialFields(t *testing.T) {
	handler := NewHandler(false, 100, 10*time.Minute)

	tests := []struct {
		name           string
		format         string
		logLine        string
		shouldSucceed  bool
		expectedError  string
		validateFields func(t *testing.T, record *FlowLogRecord)
	}{
		{
			name:          "Custom format with all V2 fields plus additional fields",
			format:        "${version} ${account-id} ${interface-id} ${srcaddr} ${dstaddr} ${srcport} ${dstport} ${protocol} ${packets} ${bytes} ${start} ${end} ${action} ${log-status} ${vpc-id} ${subnet-id}",
			logLine:       "3 123456789012 eni-abc123 10.0.1.100 192.168.1.50 443 49152 6 25 4000 1620000000 1620000060 ACCEPT OK vpc-12345 subnet-67890",
			shouldSucceed: true,
			validateFields: func(t *testing.T, record *FlowLogRecord) {
				assert.Equal(t, "3", record.Version)
				assert.Equal(t, "123456789012", record.AccountID)
				assert.Equal(t, "eni-abc123", record.InterfaceID)
				assert.Equal(t, "10.0.1.100", record.SrcAddr)
				assert.Equal(t, "vpc-12345", record.VpcID)
				assert.Equal(t, "subnet-67890", record.SubnetID)
			},
		},
		{
			name:          "Custom format with all V2 fields in different order",
			format:        "${start} ${end} ${version} ${account-id} ${interface-id} ${bytes} ${packets} ${srcaddr} ${dstaddr} ${srcport} ${dstport} ${protocol} ${action} ${log-status}",
			logLine:       "1620000000 1620000060 2 123456789012 eni-abc123 4000 25 10.0.1.100 192.168.1.50 443 49152 6 ACCEPT OK",
			shouldSucceed: true,
			validateFields: func(t *testing.T, record *FlowLogRecord) {
				assert.Equal(t, "2", record.Version)
				assert.Equal(t, "123456789012", record.AccountID)
				assert.Equal(t, int64(4000), record.Bytes)
			},
		},
		{
			name:          "Missing V2 field: account-id",
			format:        "${version} ${interface-id} ${srcaddr} ${dstaddr} ${srcport} ${dstport} ${protocol} ${packets} ${bytes} ${start} ${end} ${action} ${log-status}",
			logLine:       "3 eni-abc123 10.0.1.100 192.168.1.50 443 49152 6 25 4000 1620000000 1620000060 ACCEPT OK",
			shouldSucceed: false,
			expectedError: "Missing required field: 'account-id'",
		},
		{
			name:          "Missing V2 field: interface-id",
			format:        "${version} ${account-id} ${srcaddr} ${dstaddr} ${srcport} ${dstport} ${protocol} ${packets} ${bytes} ${start} ${end} ${action} ${log-status}",
			logLine:       "3 123456789012 10.0.1.100 192.168.1.50 443 49152 6 25 4000 1620000000 1620000060 ACCEPT OK",
			shouldSucceed: false,
			expectedError: "Missing required field: 'interface-id'",
		},
		{
			name:          "Missing V2 field: start",
			format:        "${version} ${account-id} ${interface-id} ${srcaddr} ${dstaddr} ${srcport} ${dstport} ${protocol} ${packets} ${bytes} ${end} ${action} ${log-status}",
			logLine:       "3 123456789012 eni-abc123 10.0.1.100 192.168.1.50 443 49152 6 25 4000 1620000060 ACCEPT OK",
			shouldSucceed: false,
			expectedError: "Missing required field: 'start'",
		},
		{
			name:          "Missing V2 field: end",
			format:        "${version} ${account-id} ${interface-id} ${srcaddr} ${dstaddr} ${srcport} ${dstport} ${protocol} ${packets} ${bytes} ${start} ${action} ${log-status}",
			logLine:       "3 123456789012 eni-abc123 10.0.1.100 192.168.1.50 443 49152 6 25 4000 1620000000 ACCEPT OK",
			shouldSucceed: false,
			expectedError: "Missing required field: 'end'",
		},
		{
			name:          "Missing V2 field: bytes",
			format:        "${version} ${account-id} ${interface-id} ${srcaddr} ${dstaddr} ${srcport} ${dstport} ${protocol} ${packets} ${start} ${end} ${action} ${log-status}",
			logLine:       "3 123456789012 eni-abc123 10.0.1.100 192.168.1.50 443 49152 6 25 1620000000 1620000060 ACCEPT OK",
			shouldSucceed: false,
			expectedError: "Missing required field: 'bytes'",
		},
		{
			name:          "Missing V2 field: packets",
			format:        "${version} ${account-id} ${interface-id} ${srcaddr} ${dstaddr} ${srcport} ${dstport} ${protocol} ${bytes} ${start} ${end} ${action} ${log-status}",
			logLine:       "3 123456789012 eni-abc123 10.0.1.100 192.168.1.50 443 49152 6 4000 1620000000 1620000060 ACCEPT OK",
			shouldSucceed: false,
			expectedError: "Missing required field: 'packets'",
		},
		{
			name:          "Missing V2 field: action",
			format:        "${version} ${account-id} ${interface-id} ${srcaddr} ${dstaddr} ${srcport} ${dstport} ${protocol} ${packets} ${bytes} ${start} ${end} ${log-status}",
			logLine:       "3 123456789012 eni-abc123 10.0.1.100 192.168.1.50 443 49152 6 25 4000 1620000000 1620000060 OK",
			shouldSucceed: false,
			expectedError: "Missing required field: 'action'",
		},
		{
			name:          "Missing V2 field: log-status",
			format:        "${version} ${account-id} ${interface-id} ${srcaddr} ${dstaddr} ${srcport} ${dstport} ${protocol} ${packets} ${bytes} ${start} ${end} ${action}",
			logLine:       "3 123456789012 eni-abc123 10.0.1.100 192.168.1.50 443 49152 6 25 4000 1620000000 1620000060 ACCEPT",
			shouldSucceed: false,
			expectedError: "Missing required field: 'log-status'",
		},
		{
			name:          "Custom format with all V2 fields plus ECS fields (version 8)",
			format:        "${version} ${account-id} ${interface-id} ${srcaddr} ${dstaddr} ${srcport} ${dstport} ${protocol} ${packets} ${bytes} ${start} ${end} ${action} ${log-status} ${ecs-cluster-name} ${ecs-service-name}",
			logLine:       "8 123456789012 eni-abc123 10.0.1.100 192.168.1.50 443 49152 6 25 4000 1620000000 1620000060 ACCEPT OK my-cluster my-service",
			shouldSucceed: true,
			validateFields: func(t *testing.T, record *FlowLogRecord) {
				assert.Equal(t, "8", record.Version)
				assert.Equal(t, "my-cluster", record.ECSClusterName)
				assert.Equal(t, "my-service", record.ECSServiceName)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record, err := handler.parseFlowLogRecordCustom(tt.logLine, tt.format)

			if tt.shouldSucceed {
				require.NoError(t, err, "Should parse successfully")
				require.NotNil(t, record, "Should return a valid record")

				if tt.validateFields != nil {
					tt.validateFields(t, record)
				}

				// Verify metrics can be created from this record
				metrics := handler.createMetrics(record)
				assert.NotNil(t, metrics, "Should create metrics")
				assert.Equal(t, 1, metrics.ResourceMetrics().Len(), "Should have metrics")
			} else {
				require.Error(t, err, "Should fail parsing")
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError, "Error message should match")
				}
				assert.Nil(t, record, "Should return nil record on error")
			}
		})
	}
}

// TestFieldPresenceMap tests the FieldPresenceMap functionality
func TestFieldPresenceMap(t *testing.T) {
	tests := []struct {
		name           string
		format         string
		expectedFields map[string]bool
	}{
		{
			name:           "Default format returns nil",
			format:         VpcFlowLogsDefaultFormatString,
			expectedFields: nil, // nil indicates default format
		},
		{
			name:           "Empty format returns nil",
			format:         "",
			expectedFields: nil,
		},
		{
			name:   "Custom format with few fields",
			format: "${version} ${start} ${end} ${bytes}",
			expectedFields: map[string]bool{
				"version": true,
				"start":   true,
				"end":     true,
				"bytes":   true,
			},
		},
		{
			name:   "Custom format with many fields",
			format: "${version} ${account-id} ${interface-id} ${srcaddr} ${dstaddr} ${start} ${end} ${bytes} ${packets} ${vpc-id}",
			expectedFields: map[string]bool{
				"version":      true,
				"account-id":   true,
				"interface-id": true,
				"srcaddr":      true,
				"dstaddr":      true,
				"start":        true,
				"end":          true,
				"bytes":        true,
				"packets":      true,
				"vpc-id":       true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			presence := NewFieldPresenceMap(tt.format)

			if tt.expectedFields == nil {
				assert.Nil(t, presence, "Should return nil for default/empty format")
			} else {
				require.NotNil(t, presence, "Should return non-nil map")

				// Check all expected fields are present
				for field, shouldBePresent := range tt.expectedFields {
					assert.Equal(t, shouldBePresent, presence.HasField(field),
						"Field %s presence should match", field)
				}

				// Check that non-included fields are not present
				assert.False(t, presence.HasField("non-existent-field"))
				if !presence.HasField("account-id") {
					assert.False(t, presence.HasField("account-id"), "account-id should not be present")
				}
			}
		})
	}
}

// TestCustomFormat_AttributeInsertion tests that all V2 fields plus additional fields work correctly
func TestCustomFormat_AttributeInsertion(t *testing.T) {
	handler := NewHandler(false, 100, 10*time.Minute)

	// Parse a custom format with all V2 fields plus additional fields
	format := "${version} ${account-id} ${interface-id} ${srcaddr} ${dstaddr} ${srcport} ${dstport} ${protocol} ${packets} ${bytes} ${start} ${end} ${action} ${log-status} ${vpc-id} ${region}"
	logLine := "3 123456789012 eni-abc123 10.0.1.100 192.168.1.50 443 49152 6 25 4000 1620000000 1620000060 ACCEPT OK vpc-12345 us-east-1"

	record, err := handler.parseFlowLogRecordCustom(logLine, format)
	require.NoError(t, err)
	require.NotNil(t, record)

	// Create metrics
	metrics := handler.createMetrics(record)
	require.NotNil(t, metrics)

	// Extract attributes from the first data point
	rm := metrics.ResourceMetrics().At(0)
	scopeMetrics := rm.ScopeMetrics().At(0)
	require.Greater(t, scopeMetrics.Metrics().Len(), 0, "Should have metrics")

	// Check byte metric attributes
	byteMetric := scopeMetrics.Metrics().At(0)
	require.Greater(t, byteMetric.Gauge().DataPoints().Len(), 0, "Should have data points")

	dp := byteMetric.Gauge().DataPoints().At(0)
	attrs := dp.Attributes()

	// Verify all V2 fields have attributes
	versionVal, exists := attrs.Get("version")
	assert.True(t, exists, "version should be present")
	assert.Equal(t, "3", versionVal.Str())

	accountIDVal, exists := attrs.Get("account_id")
	assert.True(t, exists, "account_id should be present")
	assert.Equal(t, "123456789012", accountIDVal.Str())

	srcAddrVal, exists := attrs.Get("src_addr")
	assert.True(t, exists, "src_addr should be present")
	assert.Equal(t, "10.0.1.100", srcAddrVal.Str())

	// Verify additional fields work
	vpcIDVal, exists := attrs.Get("vpc_id")
	assert.True(t, exists, "vpc_id should be present")
	assert.Equal(t, "vpc-12345", vpcIDVal.Str())

	regionVal, exists := attrs.Get("region")
	assert.True(t, exists, "region should be present")
	assert.Equal(t, "us-east-1", regionVal.Str())
}

// TestCustomFormat_RealWorldScenarios tests real-world custom format scenarios with all V2 fields
func TestCustomFormat_RealWorldScenarios(t *testing.T) {
	handler := NewHandler(false, 100, 10*time.Minute)

	tests := []struct {
		name        string
		format      string
		logLine     string
		description string
	}{
		{
			name:        "V2 fields with region tracking",
			format:      "${version} ${account-id} ${interface-id} ${srcaddr} ${dstaddr} ${srcport} ${dstport} ${protocol} ${packets} ${bytes} ${start} ${end} ${action} ${log-status} ${region} ${az-id}",
			logLine:     "5 123456789012 eni-abc123 10.0.1.100 192.168.1.50 443 49152 6 25 4000 1620000000 1620000060 ACCEPT OK us-east-1 use1-az1",
			description: "Customer tracking traffic across regions with all V2 fields",
		},
		{
			name:        "V2 fields with reject reason analysis",
			format:      "${version} ${account-id} ${interface-id} ${srcaddr} ${dstaddr} ${srcport} ${dstport} ${protocol} ${packets} ${bytes} ${start} ${end} ${action} ${log-status} ${reject-reason}",
			logLine:     "8 123456789012 eni-abc123 10.0.1.100 192.168.1.50 443 49152 6 0 0 1620000000 1620000060 REJECT NODATA SecurityGroupRule",
			description: "Customer wants to analyze rejected traffic with all V2 fields",
		},
		{
			name:        "V2 fields with ECS container monitoring",
			format:      "${version} ${account-id} ${interface-id} ${srcaddr} ${dstaddr} ${srcport} ${dstport} ${protocol} ${packets} ${bytes} ${start} ${end} ${action} ${log-status} ${ecs-cluster-name} ${ecs-service-name} ${ecs-task-id}",
			logLine:     "8 123456789012 eni-abc123 10.0.1.100 192.168.1.50 443 49152 6 25 4000 1620000000 1620000060 ACCEPT OK prod-cluster api-service task-abc123",
			description: "Customer monitoring ECS container traffic with all V2 fields",
		},
		{
			name:        "V2 fields with VPC and subnet tracking",
			format:      "${version} ${account-id} ${interface-id} ${srcaddr} ${dstaddr} ${srcport} ${dstport} ${protocol} ${packets} ${bytes} ${start} ${end} ${action} ${log-status} ${vpc-id} ${subnet-id} ${instance-id}",
			logLine:     "3 123456789012 eni-abc123 10.0.1.100 192.168.1.50 443 49152 6 25 4000 1620000000 1620000060 ACCEPT OK vpc-12345 subnet-67890 i-instance123",
			description: "Customer tracking VPC and subnet details with all V2 fields",
		},
		{
			name:        "Future v11 format with unknown field - parse known fields successfully",
			format:      "${version} ${account-id} ${interface-id} ${srcaddr} ${dstaddr} ${srcport} ${dstport} ${protocol} ${packets} ${bytes} ${start} ${end} ${action} ${log-status} ${connection-id}",
			logLine:     "11 123456789012 eni-abc123 10.0.1.100 192.168.1.50 443 49152 6 25 4000 1620000000 1620000060 ACCEPT OK conn-xyz789",
			description: "Forward compatibility: v11 with new unknown field should parse all known fields successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.description)

			record, err := handler.parseFlowLogRecordCustom(tt.logLine, tt.format)
			require.NoError(t, err, "Should parse successfully")
			require.NotNil(t, record, "Should return a valid record")

			// Verify metrics can be created
			metrics := handler.createMetrics(record)
			require.NotNil(t, metrics, "Should create metrics")
			assert.Equal(t, 1, metrics.ResourceMetrics().Len(), "Should have resource metrics")

			rm := metrics.ResourceMetrics().At(0)
			assert.Greater(t, rm.ScopeMetrics().Len(), 0, "Should have scope metrics")
		})
	}
}
