package vpc_flow_logs

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// TestDataCategory represents different categories of test data
type TestDataCategory int

const (
	ValidRecords TestDataCategory = iota
	InvalidVersion
	InvalidAccountID
	InvalidAction
	InvalidLogStatus
	InvalidFieldCount
	InvalidIntegerFields
	MalformedRecords
)

// loadTestData loads and categorizes VPC flow log test data from testdata file
func loadTestData(t *testing.T) map[TestDataCategory][]string {
	file, err := os.Open("../testdata/vpc_flow_log_event1.txt")
	require.NoError(t, err, "Failed to open test data file")
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			t.Errorf("Error closing file: %v", closeErr)
		}
	}()

	data := make(map[TestDataCategory][]string)
	scanner := bufio.NewScanner(file)

	currentCategory := ValidRecords

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			// Check for category markers in comments
			if strings.Contains(line, "Wrong version") {
				currentCategory = InvalidVersion
			} else if strings.Contains(line, "Invalid account IDs") {
				currentCategory = InvalidAccountID
			} else if strings.Contains(line, "Invalid actions") {
				currentCategory = InvalidAction
			} else if strings.Contains(line, "Invalid log status") {
				currentCategory = InvalidLogStatus
			} else if strings.Contains(line, "Field count errors") {
				currentCategory = InvalidFieldCount
			} else if strings.Contains(line, "Invalid integer fields") {
				currentCategory = InvalidIntegerFields
			} else if strings.Contains(line, "Empty and malformed") {
				currentCategory = MalformedRecords
			} else if strings.Contains(line, "INVALID RECORDS") {
				// Reset to continue with invalid records
				continue
			} else if strings.Contains(line, "Valid records") {
				currentCategory = ValidRecords
			}
			continue
		}

		data[currentCategory] = append(data[currentCategory], line)
	}

	require.NoError(t, scanner.Err(), "Error reading test data file")
	return data
}

func TestParseFlowLogRecord_WithTestData(t *testing.T) {
	// This comprehensive test uses external test data to validate all parsing scenarios.
	// It provides better coverage than hardcoded test cases because:
	// 1. Uses 50+ real-world VPC Flow Log examples from testdata/vpc_flow_log_event1.txt
	// 2. Covers all validation scenarios (version, account ID, action, log status, field counts)
	// 3. Easy to extend by adding new test cases to the external file
	// 4. Organized by validation categories for better test organization
	handler := &Handler{}
	testData := loadTestData(t)

	t.Run("Valid Records", func(t *testing.T) {
		validRecords := testData[ValidRecords]
		require.NotEmpty(t, validRecords, "Should have valid test records")

		for i, logData := range validRecords {
			t.Run(fmt.Sprintf("ValidRecord_%d", i), func(t *testing.T) {
				result, err := handler.parseFlowLogRecordDefault(logData)
				assert.NoError(t, err, "Valid record should parse without error: %s", logData)
				assert.NotNil(t, result, "Valid record should return a result")

				if result != nil {
					// Verify basic structure
					assert.Equal(t, VpcFlowLogsDefaultVersion, result.Version, "Version should match default version in test data")
					assert.Len(t, result.AccountID, 12, "Account ID should be 12 digits")
					assert.Contains(t, []string{"ACCEPT", "REJECT"}, result.Action, "Action should be ACCEPT or REJECT")
					assert.Contains(t, []string{"OK", "NODATA", "SKIPDATA"}, result.LogStatus, "LogStatus should be valid")
				}
			})
		}
	})

	t.Run("Invalid Version Records", func(t *testing.T) {
		invalidVersionRecords := testData[InvalidVersion]
		require.NotEmpty(t, invalidVersionRecords, "Should have invalid version test records")

		for i, logData := range invalidVersionRecords {
			t.Run(fmt.Sprintf("InvalidVersion_%d", i), func(t *testing.T) {
				result, err := handler.parseFlowLogRecordDefault(logData)

				// Version 3+ are valid (forward compatibility), only version < 2 should fail
				// Test data has: version 1 (should fail), version 3 (should succeed), "version" (non-numeric, parses as 0, should fail)
				if strings.HasPrefix(logData, "3 ") {
					// Version 3 and higher should succeed
					assert.NoError(t, err, "Version should be valid: %s", logData)
					assert.NotNil(t, result, "Should return a result")
				} else {
					// Version 1 and non-numeric versions (< 2) should fail
					assert.Error(t, err, "Invalid version record should fail: %s", logData)
					assert.Nil(t, result, "Invalid version record should return nil result")

					validationErr, ok := err.(*ValidationError)
					if ok {
						assert.Equal(t, ConvertKeyToAWSFieldName(VersionKey), validationErr.Field, "Error should be for version field")
					}
				}
			})
		}
	})

	t.Run("Invalid Account ID Records", func(t *testing.T) {
		invalidAccountRecords := testData[InvalidAccountID]
		require.NotEmpty(t, invalidAccountRecords, "Should have invalid account ID test records")

		for i, logData := range invalidAccountRecords {
			t.Run(fmt.Sprintf("InvalidAccountID_%d", i), func(t *testing.T) {
				result, err := handler.parseFlowLogRecordDefault(logData)
				assert.Error(t, err, "Invalid account ID record should fail: %s", logData)
				assert.Nil(t, result, "Invalid account ID record should return nil result")

				validationErr, ok := err.(*ValidationError)
				if ok {
					assert.Equal(t, ConvertKeyToAWSFieldName(AccountIDKey), validationErr.Field, "Error should be for account ID field")
				}
			})
		}
	})

	t.Run("Invalid Action Records", func(t *testing.T) {
		invalidActionRecords := testData[InvalidAction]
		require.NotEmpty(t, invalidActionRecords, "Should have invalid action test records")

		for i, logData := range invalidActionRecords {
			t.Run(fmt.Sprintf("InvalidAction_%d", i), func(t *testing.T) {
				result, err := handler.parseFlowLogRecordDefault(logData)
				assert.Error(t, err, "Invalid action record should fail: %s", logData)
				assert.Nil(t, result, "Invalid action record should return nil result")

				validationErr, ok := err.(*ValidationError)
				if ok {
					assert.Equal(t, ConvertKeyToAWSFieldName(ActionKey), validationErr.Field, "Error should be for action field")
				}
			})
		}
	})

	t.Run("Invalid Log Status Records", func(t *testing.T) {
		invalidLogStatusRecords := testData[InvalidLogStatus]
		require.NotEmpty(t, invalidLogStatusRecords, "Should have invalid log status test records")

		for i, logData := range invalidLogStatusRecords {
			t.Run(fmt.Sprintf("InvalidLogStatus_%d", i), func(t *testing.T) {
				result, err := handler.parseFlowLogRecordDefault(logData)
				assert.Error(t, err, "Invalid log status record should fail: %s", logData)
				assert.Nil(t, result, "Invalid log status record should return nil result")

				validationErr, ok := err.(*ValidationError)
				if ok {
					assert.Equal(t, ConvertKeyToAWSFieldName(LogStatusKey), validationErr.Field, "Error should be for log status field")
				}
			})
		}
	})

	t.Run("Invalid Field Count Records", func(t *testing.T) {
		invalidFieldCountRecords := testData[InvalidFieldCount]
		require.NotEmpty(t, invalidFieldCountRecords, "Should have invalid field count test records")

		for i, logData := range invalidFieldCountRecords {
			t.Run(fmt.Sprintf("InvalidFieldCount_%d", i), func(t *testing.T) {
				result, err := handler.parseFlowLogRecordDefault(logData)
				assert.Error(t, err, "Invalid field count record should fail: %s", logData)
				assert.Nil(t, result, "Invalid field count record should return nil result")

				parseErr, ok := err.(*ParseError)
				if ok {
					assert.Equal(t, VpcFlowLogsDefaultVersionFieldsCount, parseErr.Expected, "Error should expect correct field count")
				}
			})
		}
	})

	t.Run("Invalid Integer Fields Records", func(t *testing.T) {
		invalidIntegerRecords := testData[InvalidIntegerFields]
		require.NotEmpty(t, invalidIntegerRecords, "Should have invalid integer test records")

		for i, logData := range invalidIntegerRecords {
			t.Run(fmt.Sprintf("InvalidInteger_%d", i), func(t *testing.T) {
				// These should parse but have 0 values for invalid integers
				result, err := handler.parseFlowLogRecordDefault(logData)

				// The record might still be valid if validation passes, but integers will be 0
				if err != nil {
					// If there's an error, it should be validation error, not parse error
					_, isValidationErr := err.(*ValidationError)
					assert.True(t, isValidationErr, "Should be validation error if any error occurs")
				} else {
					assert.NotNil(t, result, "Should have result if no validation error")
					// Note: Individual integer fields may be 0 due to invalid parsing, but this is acceptable
				}
			})
		}
	})

	t.Run("Malformed Records", func(t *testing.T) {
		malformedRecords := testData[MalformedRecords]
		require.NotEmpty(t, malformedRecords, "Should have malformed test records")

		for i, logData := range malformedRecords {
			t.Run(fmt.Sprintf("Malformed_%d", i), func(t *testing.T) {
				result, err := handler.parseFlowLogRecordDefault(logData)
				assert.Error(t, err, "Malformed record should fail: %s", logData)
				assert.Nil(t, result, "Malformed record should return nil result")

				// Should be a parse error due to insufficient fields
				parseErr, ok := err.(*ParseError)
				if ok {
					assert.Equal(t, VpcFlowLogsDefaultVersionFieldsCount, parseErr.Expected, "Error should expect correct field count")
				}
			})
		}
	})
}

func TestTransformVpcFlowLogs_WithTestData(t *testing.T) {
	// This comprehensive test validates the complete TransformVpcFlowLogs workflow using external test data.
	// It provides superior coverage compared to hardcoded tests because:
	// 1. Tests mixed valid/invalid records in a single execution (realistic scenario)
	// 2. Uses comprehensive test data from testdata/vpc_flow_log_event1.txt
	// 3. Verifies error handling (invalid records are skipped gracefully)
	// 4. Validates complete metrics structure and OpenTelemetry format compliance

	handler := NewHandler(false, 100, 10*time.Minute)

	// Mock the getFlowLogFormat function to return AWS default format (avoids EC2 API call)
	handler.getFlowLogFormat = func(logGroupName string) (string, string, int, error) {
		return VpcFlowLogsDefaultFormatString, "fl-mock-12345", 1, nil
	}

	testData := loadTestData(t)
	ctx := context.Background()

	// Create mixed input with valid and invalid records
	var input []events.CloudwatchLogsLogEvent
	validRecords := testData[ValidRecords]
	invalidRecords := testData[InvalidFieldCount] // Use field count errors as they're definitely invalid

	// Add some valid records
	for i, logData := range validRecords[:3] { // Use first 3 valid records
		input = append(input, events.CloudwatchLogsLogEvent{
			ID:        fmt.Sprintf("valid-%d", i),
			Timestamp: time.Now().Unix() * 1000,
			Message:   logData,
		})
	}

	// Add some invalid records
	for i, logData := range invalidRecords[:2] { // Use first 2 invalid records
		input = append(input, events.CloudwatchLogsLogEvent{
			ID:        fmt.Sprintf("invalid-%d", i),
			Timestamp: time.Now().Unix() * 1000,
			Message:   logData,
		})
	}

	output := make(chan pmetric.Metrics, 10)

	// Execute
	handler.TransformVpcFlowLogs(ctx, "123456789012", "vpc-flow-logs", "stream1", input, output)

	// Verify results - should only get metrics for valid records
	var results []pmetric.Metrics
	for metrics := range output {
		results = append(results, metrics)
	}

	assert.Equal(t, 3, len(results), "Should get metrics only for valid records (3)")

	// Verify each result has the expected structure
	for i, metrics := range results {
		assert.Equal(t, 1, metrics.ResourceMetrics().Len(), "Result %d should have 1 resource metric", i)
		rm := metrics.ResourceMetrics().At(0)
		assert.Equal(t, 1, rm.ScopeMetrics().Len(), "Result %d should have 1 scope metric", i)
		scope := rm.ScopeMetrics().At(0)
		assert.Equal(t, 2, scope.Metrics().Len(), "Result %d should have 2 metrics (bytes and packets)", i)
	}
}

func TestNewHandler(t *testing.T) {
	tests := []struct {
		name                  string
		isDebugEnabled        bool
		fullDebugInterval     int
		expectedDebugInterval int
		expectedDebugEnabled  bool
	}{
		{
			name:                  "Debug enabled with valid interval",
			isDebugEnabled:        true,
			fullDebugInterval:     50,
			expectedDebugInterval: 50,
			expectedDebugEnabled:  true,
		},
		{
			name:                  "Debug disabled with valid interval",
			isDebugEnabled:        false,
			fullDebugInterval:     200,
			expectedDebugInterval: 200,
			expectedDebugEnabled:  false,
		},
		{
			name:                  "Debug enabled with zero interval (should use default)",
			isDebugEnabled:        true,
			fullDebugInterval:     0,
			expectedDebugInterval: 100,
			expectedDebugEnabled:  true,
		},
		{
			name:                  "Debug enabled with negative interval (should use default)",
			isDebugEnabled:        true,
			fullDebugInterval:     -10,
			expectedDebugInterval: 100,
			expectedDebugEnabled:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(tt.isDebugEnabled, tt.fullDebugInterval, 10*time.Minute)

			assert.Equal(t, tt.expectedDebugEnabled, handler.isDebugEnabled)
			assert.Equal(t, tt.expectedDebugInterval, handler.fullDebugInterval)
			assert.Equal(t, 0, handler.debugCounter)
			assert.NotNil(t, handler.formatCache)
		})
	}
}

func TestConvertKeyToAWSFieldName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Version", VersionKey, "version"},
		{"Account ID", AccountIDKey, "account-id"},
		{"Interface ID", InterfaceIDKey, "interface-id"},
		{"Source Address", SrcAddrKey, "srcaddr"},
		{"Destination Address", DstAddrKey, "dstaddr"},
		{"Source Port", SrcPortKey, "srcport"},
		{"Destination Port", DstPortKey, "dstport"},
		{"Protocol", ProtocolKey, "protocol"},
		{"Protocol Name", ProtocolNameKey, "protocolName"},
		{"Packets", PacketsKey, "packets"},
		{"Bytes", BytesKey, "bytes"},
		{"Start", StartKey, "start"},
		{"End", EndKey, "end"},
		{"Action", ActionKey, "action"},
		{"Log Status", LogStatusKey, "log-status"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertKeyToAWSFieldName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertProtocol(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"ICMP", "1", "ICMP"},
		{"TCP", "6", "TCP"},
		{"UDP", "17", "UDP"},
		{"GRE", "47", "GRE"},
		{"ESP", "50", "ESP"},
		{"AH", "51", "AH"},
		{"ICMPv6", "58", "ICMPv6"},
		{"OSPF", "89", "OSPF"},
		{"SCTP", "132", "SCTP"},
		{"Unknown Protocol", "255", "255"},
		{"Empty String", "", ""},
		{"Non-numeric", "abc", "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertProtocol(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseToStruct tests the parseToStruct function
func TestParseToStruct(t *testing.T) {
	tests := []struct {
		name        string
		format      string
		line        string
		expectedErr bool
		validate    func(*testing.T, *FlowLogRecord)
	}{
		{
			name:        "Valid AWS default format",
			format:      "${version} ${account-id} ${interface-id} ${srcaddr} ${dstaddr} ${srcport} ${dstport} ${protocol} ${packets} ${bytes} ${start} ${end} ${action} ${log-status}",
			line:        "2 123456789012 eni-1235b8ca123456789 172.31.16.139 172.31.16.21 20641 22 6 20 4249 1418530010 1418530070 ACCEPT OK",
			expectedErr: false,
			validate: func(t *testing.T, record *FlowLogRecord) {
				assert.Equal(t, "2", record.Version)
				assert.Equal(t, "123456789012", record.AccountID)
				assert.Equal(t, "eni-1235b8ca123456789", record.InterfaceID)
				assert.Equal(t, "172.31.16.139", record.SrcAddr)
				assert.Equal(t, "172.31.16.21", record.DstAddr)
				assert.Equal(t, "20641", record.SrcPort)
				assert.Equal(t, "22", record.DstPort)
				assert.Equal(t, "6", record.Protocol)
				assert.Equal(t, int64(20), record.Packets)
				assert.Equal(t, int64(4249), record.Bytes)
				assert.Equal(t, int64(1418530010), record.Start)
				assert.Equal(t, int64(1418530070), record.End)
				assert.Equal(t, "ACCEPT", record.Action)
				assert.Equal(t, "OK", record.LogStatus)
			},
		},
		{
			name:        "Valid custom format with additional fields",
			format:      "${version} ${account-id} ${vpc-id} ${subnet-id} ${srcaddr} ${dstaddr} ${action}",
			line:        "3 123456789012 vpc-12345678 subnet-87654321 10.0.1.5 10.0.2.10 ACCEPT",
			expectedErr: false,
			validate: func(t *testing.T, record *FlowLogRecord) {
				assert.Equal(t, "3", record.Version)
				assert.Equal(t, "123456789012", record.AccountID)
				assert.Equal(t, "vpc-12345678", record.VpcID)
				assert.Equal(t, "subnet-87654321", record.SubnetID)
				assert.Equal(t, "10.0.1.5", record.SrcAddr)
				assert.Equal(t, "10.0.2.10", record.DstAddr)
				assert.Equal(t, "ACCEPT", record.Action)
			},
		},
		{
			name:        "Field count mismatch - too many fields in line",
			format:      "${version} ${account-id} ${action}",
			line:        "2 123456789012 ACCEPT EXTRA_FIELD",
			expectedErr: true,
		},
		{
			name:        "Field count mismatch - too few fields in line",
			format:      "${version} ${account-id} ${interface-id} ${action}",
			line:        "2 123456789012",
			expectedErr: true,
		},
		{
			name:        "Mixed field types with int64 parsing",
			format:      "${packets} ${bytes} ${start} ${end}",
			line:        "100 5000 1418530010 1418530070",
			expectedErr: false,
			validate: func(t *testing.T, record *FlowLogRecord) {
				assert.Equal(t, int64(100), record.Packets)
				assert.Equal(t, int64(5000), record.Bytes)
				assert.Equal(t, int64(1418530010), record.Start)
				assert.Equal(t, int64(1418530070), record.End)
			},
		},
		{
			name:        "Invalid int64 values default to 0",
			format:      "${packets} ${bytes}",
			line:        "invalid_number not_a_number",
			expectedErr: false,
			validate: func(t *testing.T, record *FlowLogRecord) {
				assert.Equal(t, int64(0), record.Packets)
				assert.Equal(t, int64(0), record.Bytes)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record, err := parseToStruct(tt.format, tt.line, false) // isDebugEnabled=false for tests

			if tt.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, record)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, record)
				if tt.validate != nil {
					tt.validate(t, record)
				}
			}
		})
	}
}

// TestValidateFlowLogRecord tests the enhanced validateFlowLogRecord function
func TestValidateFlowLogRecord(t *testing.T) {
	handler := NewHandler(false, 100, 10*time.Minute)

	tests := []struct {
		name        string
		record      *FlowLogRecord
		expectedErr bool
		errorField  string
	}{
		{
			name: "Valid record",
			record: &FlowLogRecord{
				Version:     "2",
				AccountID:   "123456789012",
				InterfaceID: "eni-1235b8ca123456789",
				SrcAddr:     "172.31.16.139",
				DstAddr:     "172.31.16.21",
				SrcPort:     "20641",
				DstPort:     "22",
				Protocol:    "6",
				Packets:     20,
				Bytes:       4249,
				Start:       1418530010,
				End:         1418530070,
				Action:      "ACCEPT",
				LogStatus:   "OK",
			},
			expectedErr: false,
		},
		{
			name: "Empty version",
			record: &FlowLogRecord{
				Version:     "",
				AccountID:   "123456789012",
				InterfaceID: "eni-1235b8ca123456789",
				SrcAddr:     "172.31.16.139",
				DstAddr:     "172.31.16.21",
				SrcPort:     "20641",
				DstPort:     "22",
				Protocol:    "6",
				Action:      "ACCEPT",
				LogStatus:   "OK",
			},
			expectedErr: true,
			errorField:  "version",
		},
		{
			name: "Empty account ID",
			record: &FlowLogRecord{
				Version:     "2",
				AccountID:   "",
				InterfaceID: "eni-1235b8ca123456789",
				SrcAddr:     "172.31.16.139",
				DstAddr:     "172.31.16.21",
				SrcPort:     "20641",
				DstPort:     "22",
				Protocol:    "6",
				Action:      "ACCEPT",
				LogStatus:   "OK",
			},
			expectedErr: true,
			errorField:  "account-id",
		},
		{
			name: "Invalid account ID length",
			record: &FlowLogRecord{
				Version:     "2",
				AccountID:   "12345",
				InterfaceID: "eni-1235b8ca123456789",
				SrcAddr:     "172.31.16.139",
				DstAddr:     "172.31.16.21",
				SrcPort:     "20641",
				DstPort:     "22",
				Protocol:    "6",
				Action:      "ACCEPT",
				LogStatus:   "OK",
			},
			expectedErr: true,
			errorField:  "account-id",
		},
		{
			name: "Account ID with non-digits",
			record: &FlowLogRecord{
				Version:     "2",
				AccountID:   "12345678901a",
				InterfaceID: "eni-1235b8ca123456789",
				SrcAddr:     "172.31.16.139",
				DstAddr:     "172.31.16.21",
				SrcPort:     "20641",
				DstPort:     "22",
				Protocol:    "6",
				Action:      "ACCEPT",
				LogStatus:   "OK",
			},
			expectedErr: true,
			errorField:  "account-id",
		},
		{
			name: "Negative packets",
			record: &FlowLogRecord{
				Version:     "2",
				AccountID:   "123456789012",
				InterfaceID: "eni-1235b8ca123456789",
				SrcAddr:     "172.31.16.139",
				DstAddr:     "172.31.16.21",
				SrcPort:     "20641",
				DstPort:     "22",
				Protocol:    "6",
				Packets:     -1,
				Bytes:       4249,
				Start:       1418530010,
				End:         1418530070,
				Action:      "ACCEPT",
				LogStatus:   "OK",
			},
			expectedErr: true,
			errorField:  "packets",
		},
		{
			name: "Negative bytes",
			record: &FlowLogRecord{
				Version:     "2",
				AccountID:   "123456789012",
				InterfaceID: "eni-1235b8ca123456789",
				SrcAddr:     "172.31.16.139",
				DstAddr:     "172.31.16.21",
				SrcPort:     "20641",
				DstPort:     "22",
				Protocol:    "6",
				Packets:     20,
				Bytes:       -1,
				Start:       1418530010,
				End:         1418530070,
				Action:      "ACCEPT",
				LogStatus:   "OK",
			},
			expectedErr: true,
			errorField:  "bytes",
		},
		{
			name: "Start time greater than end time",
			record: &FlowLogRecord{
				Version:     "2",
				AccountID:   "123456789012",
				InterfaceID: "eni-1235b8ca123456789",
				SrcAddr:     "172.31.16.139",
				DstAddr:     "172.31.16.21",
				SrcPort:     "20641",
				DstPort:     "22",
				Protocol:    "6",
				Packets:     20,
				Bytes:       4249,
				Start:       1418530070,
				End:         1418530010,
				Action:      "ACCEPT",
				LogStatus:   "OK",
			},
			expectedErr: true,
			errorField:  "start",
		},
		{
			name: "Invalid action",
			record: &FlowLogRecord{
				Version:     "2",
				AccountID:   "123456789012",
				InterfaceID: "eni-1235b8ca123456789",
				SrcAddr:     "172.31.16.139",
				DstAddr:     "172.31.16.21",
				SrcPort:     "20641",
				DstPort:     "22",
				Protocol:    "6",
				Packets:     20,
				Bytes:       4249,
				Start:       1418530010,
				End:         1418530070,
				Action:      "INVALID",
				LogStatus:   "OK",
			},
			expectedErr: true,
			errorField:  "action",
		},
		{
			name: "Invalid log status",
			record: &FlowLogRecord{
				Version:     "2",
				AccountID:   "123456789012",
				InterfaceID: "eni-1235b8ca123456789",
				SrcAddr:     "172.31.16.139",
				DstAddr:     "172.31.16.21",
				SrcPort:     "20641",
				DstPort:     "22",
				Protocol:    "6",
				Packets:     20,
				Bytes:       4249,
				Start:       1418530010,
				End:         1418530070,
				Action:      "ACCEPT",
				LogStatus:   "INVALID",
			},
			expectedErr: true,
			errorField:  "log-status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Pass nil for default format (all fields required)
			err := handler.validateFlowLogRecord(tt.record, nil)

			if tt.expectedErr {
				assert.Error(t, err)
				if tt.errorField != "" {
					validationErr, ok := err.(*ValidationError)
					assert.True(t, ok, "Expected ValidationError")
					if ok {
						assert.Equal(t, tt.errorField, validationErr.Field)
					}
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHandler_InstrumentationScopeInMetrics(t *testing.T) {
	// This test verifies that the instrumentation scope is properly set
	// in the metrics created from VPC flow logs (integration test)
	handler := NewHandler(false, 100, 10*time.Minute)

	// Sample VPC flow log record (valid AWS default format)
	validLogMessage := "2 123456789012 eni-0a1b2c3d 10.0.1.100 192.168.1.50 443 49152 6 25 4000 1620000000 1620000060 ACCEPT OK"

	// Parse the log record
	logRecord, err := handler.parseFlowLogRecordDefault(validLogMessage)
	require.NoError(t, err, "Should parse valid VPC flow log")
	require.NotNil(t, logRecord, "Should return a valid log record")

	// Create metrics from the log record
	metrics := handler.createMetrics(logRecord)
	require.NotNil(t, metrics, "Should create metrics")

	// Verify that metrics were created
	assert.Equal(t, 1, metrics.ResourceMetrics().Len(), "Should have 1 resource metric")

	rm := metrics.ResourceMetrics().At(0)
	assert.Equal(t, 1, rm.ScopeMetrics().Len(), "Should have 1 scope metric")

	scopeMetrics := rm.ScopeMetrics().At(0)
	instrumentationScope := scopeMetrics.Scope()

	// Verify instrumentation scope is set correctly
	assert.Equal(t, "vpc_flow_logs", instrumentationScope.Name(), "Scope name should be set to vpc_flow_logs")
	assert.Equal(t, "1.0.0", instrumentationScope.Version(), "Scope version should be set to 1.0.0")

	// Verify scope attributes
	scopeAttrs := instrumentationScope.Attributes()
	assert.Equal(t, 2, scopeAttrs.Len(), "Should have 2 scope attributes")

	// Check identifier attribute
	identifierValue, exists := scopeAttrs.Get("identifier")
	assert.True(t, exists, "Identifier attribute should exist")
	assert.Equal(t, "nio", identifierValue.Str(), "Identifier should be 'nio'")

	// Check swi-reporter attribute
	swiReporterValue, exists := scopeAttrs.Get("swi-reporter")
	assert.True(t, exists, "swi-reporter attribute should exist")
	assert.Equal(t, "", swiReporterValue.Str(), "swi-reporter should be empty string")

	// Verify that metrics contain the expected data (basic check)
	assert.Equal(t, 2, scopeMetrics.Metrics().Len(), "Should have 2 metrics (bytes and packets)")
}
