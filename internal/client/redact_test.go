package client

import (
	"strings"
	"testing"
)

func TestRedactSensitiveJSON(t *testing.T) {
	in := []byte(`{"name":"x","properties":{"aws_secret_key":"sekrit","aws_access_key":"AKI","aws_bucket":"b"}}`)
	out := redactSensitiveJSON(in)
	if strings.Contains(out, "sekrit") || strings.Contains(out, "AKI") {
		t.Fatalf("secrets leaked: %s", out)
	}
	if !strings.Contains(out, `"aws_bucket":"b"`) {
		t.Fatalf("non-secret redacted: %s", out)
	}
	if !strings.Contains(out, `"***"`) {
		t.Fatalf("expected redaction markers: %s", out)
	}
}
