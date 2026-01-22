package vpc_flow_logs

import (
	"testing"
	"time"
)

// Benchmark comparing default parser vs custom parser on default format logs
// This proves that the default parser is significantly faster for default format logs

var (
	// Sample default format log lines
	defaultFormatLog1 = "2 123456789012 eni-1235b8ca123456789 172.31.16.139 172.31.16.21 20641 22 6 20 4249 1418530010 1418530070 ACCEPT OK"
	defaultFormatLog2 = "2 987654321098 eni-9876543210abcdef0 10.0.0.1 10.0.0.2 8000 8001 6 30 15000 1418530090 1418530150 ACCEPT OK"
	defaultFormatLog3 = "2 111111111111 eni-1111111111111111 192.168.0.1 192.168.0.2 443 50000 6 75 37500 1418530095 1418530155 REJECT OK"
)

// BenchmarkDefaultParser_DefaultFormat benchmarks parsing default format with the optimized default parser
func BenchmarkDefaultParser_DefaultFormat(b *testing.B) {
	handler := NewHandler(false, 100, 10*time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Use different log lines to avoid any caching effects
		logLine := defaultFormatLog1
		if i%3 == 1 {
			logLine = defaultFormatLog2
		} else if i%3 == 2 {
			logLine = defaultFormatLog3
		}

		_, err := handler.parseFlowLogRecordDefault(logLine)
		if err != nil {
			b.Fatalf("Failed to parse: %v", err)
		}
	}
}

// BenchmarkCustomParser_DefaultFormat benchmarks parsing default format with the generic custom parser
func BenchmarkCustomParser_DefaultFormat(b *testing.B) {
	handler := NewHandler(false, 100, 10*time.Minute)
	format := VpcFlowLogsDefaultFormatString

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Use different log lines to avoid any caching effects
		logLine := defaultFormatLog1
		if i%3 == 1 {
			logLine = defaultFormatLog2
		} else if i%3 == 2 {
			logLine = defaultFormatLog3
		}

		_, err := handler.parseFlowLogRecordCustom(logLine, format)
		if err != nil {
			b.Fatalf("Failed to parse: %v", err)
		}
	}
}

// BenchmarkDefaultParser_ParallelDefaultFormat benchmarks default parser under parallel load
func BenchmarkDefaultParser_ParallelDefaultFormat(b *testing.B) {
	handler := NewHandler(false, 100, 10*time.Minute)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			logLine := defaultFormatLog1
			if i%3 == 1 {
				logLine = defaultFormatLog2
			} else if i%3 == 2 {
				logLine = defaultFormatLog3
			}
			i++

			_, err := handler.parseFlowLogRecordDefault(logLine)
			if err != nil {
				b.Fatalf("Failed to parse: %v", err)
			}
		}
	})
}

// BenchmarkCustomParser_ParallelDefaultFormat benchmarks custom parser under parallel load
func BenchmarkCustomParser_ParallelDefaultFormat(b *testing.B) {
	handler := NewHandler(false, 100, 10*time.Minute)
	format := VpcFlowLogsDefaultFormatString

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			logLine := defaultFormatLog1
			if i%3 == 1 {
				logLine = defaultFormatLog2
			} else if i%3 == 2 {
				logLine = defaultFormatLog3
			}
			i++

			_, err := handler.parseFlowLogRecordCustom(logLine, format)
			if err != nil {
				b.Fatalf("Failed to parse: %v", err)
			}
		}
	})
}

// BenchmarkDefaultParser_FullPipeline benchmarks the complete pipeline with default parser
func BenchmarkDefaultParser_FullPipeline(b *testing.B) {
	handler := NewHandler(false, 100, 10*time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logLine := defaultFormatLog1
		if i%3 == 1 {
			logLine = defaultFormatLog2
		} else if i%3 == 2 {
			logLine = defaultFormatLog3
		}

		record, err := handler.parseFlowLogRecordDefault(logLine)
		if err != nil {
			b.Fatalf("Failed to parse: %v", err)
		}

		// Include metrics creation in the benchmark (real-world scenario)
		metrics := handler.createMetrics(record)
		if metrics.ResourceMetrics().Len() == 0 {
			b.Fatal("Failed to create metrics")
		}
	}
}

// BenchmarkCustomParser_FullPipeline benchmarks the complete pipeline with custom parser
func BenchmarkCustomParser_FullPipeline(b *testing.B) {
	handler := NewHandler(false, 100, 10*time.Minute)
	format := VpcFlowLogsDefaultFormatString

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logLine := defaultFormatLog1
		if i%3 == 1 {
			logLine = defaultFormatLog2
		} else if i%3 == 2 {
			logLine = defaultFormatLog3
		}

		record, err := handler.parseFlowLogRecordCustom(logLine, format)
		if err != nil {
			b.Fatalf("Failed to parse: %v", err)
		}

		// Include metrics creation in the benchmark (real-world scenario)
		metrics := handler.createMetrics(record)
		if metrics.ResourceMetrics().Len() == 0 {
			b.Fatal("Failed to create metrics")
		}
	}
}

// BenchmarkDefaultParser_HighThroughput simulates high-throughput scenario (10k records)
func BenchmarkDefaultParser_HighThroughput(b *testing.B) {
	handler := NewHandler(false, 100, 10*time.Minute)
	logs := []string{defaultFormatLog1, defaultFormatLog2, defaultFormatLog3}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10000; j++ {
			_, err := handler.parseFlowLogRecordDefault(logs[j%3])
			if err != nil {
				b.Fatalf("Failed to parse: %v", err)
			}
		}
	}
}

// BenchmarkCustomParser_HighThroughput simulates high-throughput scenario (10k records)
func BenchmarkCustomParser_HighThroughput(b *testing.B) {
	handler := NewHandler(false, 100, 10*time.Minute)
	format := VpcFlowLogsDefaultFormatString
	logs := []string{defaultFormatLog1, defaultFormatLog2, defaultFormatLog3}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10000; j++ {
			_, err := handler.parseFlowLogRecordCustom(logs[j%3], format)
			if err != nil {
				b.Fatalf("Failed to parse: %v", err)
			}
		}
	}
}
