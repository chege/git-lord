package gitcmd

import "testing"

func TestNormalizeNumstatPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "plain path", path: "internal/processor/hotspot.go", want: "internal/processor/hotspot.go"},
		{name: "simple rename", path: "old/name.go => new/name.go", want: "new/name.go"},
		{name: "brace rename", path: "internal/{old => new}/hotspot.go", want: "internal/new/hotspot.go"},
		{name: "root brace rename", path: "{old.go => new.go}", want: "new.go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeNumstatPath(tt.path); got != tt.want {
				t.Fatalf("normalizeNumstatPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
