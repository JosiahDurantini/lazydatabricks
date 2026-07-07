package bundle

import "testing"

func TestExtractRunID(t *testing.T) {
	tests := []struct {
		name string
		line string
		want int64
	}{
		{"run URL", "View run at https://adb-123.azuredatabricks.net/#job/9/runs/12345", 12345},
		{"singular run URL", "https://workspace/jobs/9/run/777", 777},
		{"run_id key-value", "run_id: 424242", 424242},
		{"run id with spaces", "Started run id  555", 555},
		{"no match", "Deploying bundle...", 0},
		{"empty line", "", 0},
		{"zero id ignored", "run_id: 0", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractRunID(tt.line); got != tt.want {
				t.Errorf("ExtractRunID(%q) = %d, want %d", tt.line, got, tt.want)
			}
		})
	}
}
