package security

import (
	"encoding/json"
	"testing"
)

func TestDefaultProfileJSON_IsValidJSON(t *testing.T) {
	b, err := DefaultProfileJSON()
	if err != nil {
		t.Fatalf("DefaultProfileJSON() returned error: %v", err)
	}
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("DefaultProfileJSON() produced invalid JSON: %v", err)
	}
}

func TestDefaultProfileJSON_DeniedSyscallsPresent(t *testing.T) {
	b, err := DefaultProfileJSON()
	if err != nil {
		t.Fatalf("DefaultProfileJSON() returned error: %v", err)
	}

	var profile struct {
		DefaultAction string `json:"defaultAction"`
		Syscalls      []struct {
			Names    []string `json:"names"`
			Action   string   `json:"action"`
			ErrnoRet uint     `json:"errnoRet"`
		} `json:"syscalls"`
	}
	if err := json.Unmarshal(b, &profile); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if profile.DefaultAction != "SCMP_ACT_ALLOW" {
		t.Errorf("expected defaultAction SCMP_ACT_ALLOW, got %s", profile.DefaultAction)
	}

	// Build a flat set of denied syscalls; also verify errnoRet is always EPERM.
	denied := map[string]bool{}
	for _, rule := range profile.Syscalls {
		if rule.Action == "SCMP_ACT_ERRNO" {
			if rule.ErrnoRet != 1 {
				t.Errorf("rule for %v has errnoRet=%d, want 1 (EPERM)", rule.Names, rule.ErrnoRet)
			}
			for _, name := range rule.Names {
				denied[name] = true
			}
		}
	}

	mustDeny := []string{"mount", "ptrace", "init_module", "unshare", "reboot"}
	for _, sc := range mustDeny {
		if !denied[sc] {
			t.Errorf("expected syscall %q to be denied, but it was not found in profile", sc)
		}
	}
}

func TestLoadOrDefault_EmptyPath(t *testing.T) {
	b, err := LoadOrDefault("")
	if err != nil {
		t.Fatalf("LoadOrDefault(\"\") error: %v", err)
	}
	if len(b) == 0 {
		t.Error("LoadOrDefault(\"\") returned empty bytes")
	}
}

func TestLoadOrDefault_MissingFile(t *testing.T) {
	_, err := LoadOrDefault("/nonexistent/path/to/profile.json")
	if err == nil {
		t.Error("expected error loading missing file, got nil")
	}
}
