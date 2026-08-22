package controllers

import "testing"

func TestAuditGroupStatus(t *testing.T) {
	tests := []struct {
		name       string
		successful int
		failed     int
		want       string
	}{
		{name: "completed", successful: 3, want: "completed"},
		{name: "partial", successful: 2, failed: 1, want: "partial"},
		{name: "failed", failed: 3, want: "failed"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := auditGroupStatus(test.successful, test.failed); got != test.want {
				t.Fatalf("auditGroupStatus(%d, %d) = %q, want %q", test.successful, test.failed, got, test.want)
			}
		})
	}
}
