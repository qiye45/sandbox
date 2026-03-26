package security

import (
	"testing"
)

func TestParseMemoryBytes(t *testing.T) {
	tests := []struct {
		input   string
		want    int64
		wantErr bool
	}{
		{"", 0, false},
		{"0", 0, false},
		{"1024", 1024, false},
		{"512MB", 512 << 20, false},
		{"4GB", 4 << 30, false},
		{"1TB", 1 << 40, false},
		{"2GiB", 2 << 30, false},
		{"1.5GB", int64(1.5 * float64(1<<30)), false},
		{"badvalue", 0, true},
	}

	for _, tc := range tests {
		got, err := ParseMemoryBytes(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseMemoryBytes(%q): expected error, got nil", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseMemoryBytes(%q): unexpected error: %v", tc.input, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseMemoryBytes(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

func TestBuildResources_PidsLimit(t *testing.T) {
	r := BuildResources(ResourceLimitsConfig{PidsLimit: 512})
	if r.PidsLimit == nil {
		t.Fatal("expected non-nil PidsLimit")
	}
	if *r.PidsLimit != 512 {
		t.Errorf("PidsLimit = %d, want 512", *r.PidsLimit)
	}
}

func TestBuildResources_NegativePids(t *testing.T) {
	r := BuildResources(ResourceLimitsConfig{PidsLimit: -1})
	if r.PidsLimit == nil {
		t.Fatal("expected non-nil PidsLimit")
	}
	if *r.PidsLimit != 0 {
		t.Errorf("negative PidsLimit should clamp to 0, got %d", *r.PidsLimit)
	}
}

func TestBuildResources_Memory(t *testing.T) {
	r := BuildResources(ResourceLimitsConfig{MemoryBytes: 4 << 30})
	if r.Memory != 4<<30 {
		t.Errorf("Memory = %d, want %d", r.Memory, 4<<30)
	}
}
