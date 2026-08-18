package parser

import (
	"fmt"
	"testing"
)

func BenchmarkParseJSON(b *testing.B) {
	p := New()
	input := `{"time":"2024-01-15T10:30:00Z","level":"ERROR","message":"connection failed","service":"api","host":"web-01"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Parse(input, "bench")
	}
}

func BenchmarkParseSyslog(b *testing.B) {
	p := New()
	input := "Jan 15 10:30:00 myhost sshd[1234]: Accepted publickey for user from 10.0.0.1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Parse(input, "bench")
	}
}

func BenchmarkParseLeveled(b *testing.B) {
	p := New()
	input := "2024-01-15 10:30:00 ERROR Something broke in the system"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Parse(input, "bench")
	}
}

func BenchmarkParseNginx(b *testing.B) {
	p := New()
	input := `192.168.1.1 - - [15/Jan/2024:10:30:00 +0000] "GET /api/health HTTP/1.1" 200 42`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Parse(input, "bench")
	}
}

func BenchmarkParseGeneric(b *testing.B) {
	p := New()
	input := "ERROR: database connection failed after 30s timeout"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Parse(input, "bench")
	}
}

func BenchmarkParseMixed(b *testing.B) {
	p := New()
	inputs := []string{
		`{"time":"2024-01-15T10:30:00Z","level":"ERROR","message":"json log"}`,
		"Jan 15 10:30:00 host syslog message",
		"2024-01-15 10:30:00 ERROR leveled log",
		`192.168.1.1 - - [15/Jan/2024:10:30:00 +0000] "GET / HTTP/1.1" 200 42`,
		"ERROR: generic log message",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Parse(inputs[i%len(inputs)], "bench")
	}
}

func BenchmarkParseHighVolume(b *testing.B) {
	p := New()
	messages := make([]string, 1000)
	for i := range messages {
		messages[i] = fmt.Sprintf(`{"time":"2024-01-15T10:30:00Z","level":"%s","message":"log entry %d"}`,
			[]string{"INFO", "WARN", "ERROR"}[i%3], i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Parse(messages[i%1000], "bench")
	}
}
