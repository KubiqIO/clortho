package service

import (
	"strings"
	"testing"
)

func TestParseCharset(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{"abc", "abc", false},
		{"a-c", "abc", false},
		{"1-3", "123", false},
		{"a-c,1-3", "abc123", false},
		{"A-C", "ABC", false},
		{"a-c,x", "abcx", false},
		{"z-a", "", true},
	}

	for _, tt := range tests {
		got, err := ParseCharset(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseCharset(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.expected {
			t.Errorf("ParseCharset(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}

	got, err := ParseCharset("")
	if err != nil {
		t.Errorf("ParseCharset(\"\") unexpected error: %v", err)
	}
	if len(got) < 10 {
		t.Errorf("ParseCharset(\"\") returned short string: %s", got)
	}
}

func TestGenerateLicenseKey(t *testing.T) {
	key, err := GenerateLicenseKey("TEST", 16, "-", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(key, "TEST-") {
		t.Errorf("expected prefix TEST-, got %s", key)
	}

	if len(key) != 21 {
		t.Errorf("expected length 21, got %d", len(key))
	}

	key2, _ := GenerateLicenseKey("", 10, "", "")
	if len(key2) != 10 {
		t.Errorf("expected length 10, got %d", len(key2))
	}

	key3, _ := GenerateLicenseKey("COOL", 8, "#", "")
	if !strings.HasPrefix(key3, "COOL#") {
		t.Errorf("expected prefix COOL#, got %s", key3)
	}
	if len(key3) != 13 {
		t.Errorf("expected length 13, got %d", len(key3))
	}

	charset := "A"
	key4, _ := GenerateLicenseKey("", 10, "", charset)
	if key4 != "AAAAAAAAAA" {
		t.Errorf("expected all As, got %s", key4)
	}

	parsed, _ := ParseCharset("1-1")
	key5, _ := GenerateLicenseKey("", 5, "", parsed)
	if key5 != "11111" {
		t.Errorf("expected all 1s, got %s", key5)
	}
}

func TestValidateLicenseKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"owned prefix", "Owned-a1b2c3d4e5f6", false},
		{"leased prefix", "Leased-a1b2c3d4e5f6", false},
		{"trial prefix", "Trial-a1b2c3d4e5f6", false},
		{"no prefix", "a1b2c3d4e5f6", false},
		{"punctuation and mixed case", "PB_2019.Owned-XyZ/09+abc=", false},
		{"single character", "x", false},
		{"at the length limit", strings.Repeat("k", MaxLicenseKeyLength), false},

		{"empty", "", true},
		{"one over the length limit", strings.Repeat("k", MaxLicenseKeyLength+1), true},
		// Keys are sent as an X-License-Key header value, so these are injection
		// vectors rather than merely unusual keys.
		{"embedded space", "Owned-a1b2 c3d4", true},
		{"leading space", " Owned-a1b2c3d4", true},
		{"trailing newline", "Owned-a1b2c3d4\n", true},
		{"carriage return", "Owned-a1b2\r\nX-Injected: 1", true},
		{"tab", "Owned-a1b2\tc3d4", true},
		{"null byte", "Owned-a1b2\x00c3d4", true},
		{"non-ascii", "Owned-café", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLicenseKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateLicenseKey(%q) error = %v, wantErr %v", tt.key, err, tt.wantErr)
			}
		})
	}
}

// Generated keys must satisfy the same rules imported ones do, or a key Clortho
// minted itself could not be re-imported into another instance.
func TestGeneratedKeysAreValidLicenseKeys(t *testing.T) {
	for _, prefix := range []string{"", "TEST", "Owned"} {
		key, err := GenerateLicenseKey(prefix, 16, "-", "")
		if err != nil {
			t.Fatalf("GenerateLicenseKey(%q) unexpected error: %v", prefix, err)
		}
		if err := ValidateLicenseKey(key); err != nil {
			t.Errorf("generated key %q failed validation: %v", key, err)
		}
	}
}
