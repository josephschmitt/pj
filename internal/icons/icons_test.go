package icons

import "testing"

func TestNewMapper(t *testing.T) {
	iconMap := map[string]string{
		".git":    "",
		"go.mod":  "󰟓",
	}

	mapper := NewMapper(iconMap)

	// Test that icons are copied (not referenced)
	iconMap[".git"] = "different"
	if mapper.Get(".git") == "different" {
		t.Error("Mapper should not be affected by changes to original map")
	}
}

func TestGet(t *testing.T) {
	tests := []struct {
		name     string
		iconMap  map[string]string
		marker   string
		expected string
	}{
		{
			name:     "existing marker",
			iconMap:  map[string]string{".git": ""},
			marker:   ".git",
			expected: "",
		},
		{
			name:     "non-existent marker returns default",
			iconMap:  map[string]string{".git": ""},
			marker:   "unknown",
			expected: "",
		},
		{
			name:     "empty map returns default",
			iconMap:  map[string]string{},
			marker:   ".git",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapper := NewMapper(tt.iconMap)
			got := mapper.Get(tt.marker)
			if got != tt.expected {
				t.Errorf("Get(%q) = %q, want %q", tt.marker, got, tt.expected)
			}
		})
	}
}

func TestSet(t *testing.T) {
	mapper := NewMapper(map[string]string{
		".git": "",
	})

	// Test updating existing
	mapper.Set(".git", "new")
	if got := mapper.Get(".git"); got != "new" {
		t.Errorf("After Set(.git, new), Get(.git) = %q, want %q", got, "new")
	}

	// Test adding new
	mapper.Set("go.mod", "󰟓")
	if got := mapper.Get("go.mod"); got != "󰟓" {
		t.Errorf("After Set(go.mod, 󰟓), Get(go.mod) = %q, want %q", got, "󰟓")
	}
}

func TestNewMapper_NilMap(t *testing.T) {
	// Verify nil input doesn't panic
	mapper := NewMapper(nil)
	if mapper == nil {
		t.Fatal("NewMapper(nil) returned nil mapper")
	}

	// Should return default icon for any marker
	if got := mapper.Get("test"); got != "" {
		t.Errorf("Get(test) on nil map = %q, want %q", got, "")
	}
}

func TestGet_EmptyMarker(t *testing.T) {
	mapper := NewMapper(map[string]string{
		"":    "empty-icon",
		".git": "",
	})

	// Empty string marker should work
	if got := mapper.Get(""); got != "empty-icon" {
		t.Errorf("Get(\"\") = %q, want %q", got, "empty-icon")
	}
}

func TestGet_UnicodeMarkers(t *testing.T) {
	tests := []struct {
		name     string
		marker   string
		icon     string
	}{
		{
			name:   "emoji marker",
			marker: "🚀.txt",
			icon:   "🎯",
		},
		{
			name:   "unicode marker",
			marker: "文件.txt",
			icon:   "📄",
		},
		{
			name:   "emoji icon",
			marker: ".git",
			icon:   "✨",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapper := NewMapper(map[string]string{
				tt.marker: tt.icon,
			})
			got := mapper.Get(tt.marker)
			if got != tt.icon {
				t.Errorf("Get(%q) = %q, want %q", tt.marker, got, tt.icon)
			}
		})
	}
}
