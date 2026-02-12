package icons

import "testing"

func TestNewMapper(t *testing.T) {
	iconMap := map[string]string{
		".git":    "",
		"go.mod":  "󰟓",
	}
	colorMap := map[string]string{
		".git": "red",
	}

	mapper := NewMapper(iconMap, colorMap, nil, nil)

	// Test that icons are copied (not referenced)
	iconMap[".git"] = "different"
	if mapper.Get(".git") == "different" {
		t.Error("Mapper should not be affected by changes to original icon map")
	}

	// Test that colors are copied (not referenced)
	colorMap[".git"] = "green"
	if mapper.GetColor(".git") == "green" {
		t.Error("Mapper should not be affected by changes to original color map")
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
			mapper := NewMapper(tt.iconMap, nil, nil, nil)
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
	}, nil, nil, nil)

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
	mapper := NewMapper(nil, nil, nil, nil)
	if mapper == nil {
		t.Fatal("NewMapper(nil, nil, nil, nil) returned nil mapper")
	}

	// Should return default icon for any marker
	if got := mapper.Get("test"); got != "" {
		t.Errorf("Get(test) on nil map = %q, want %q", got, "")
	}

	// Should return default color for any marker
	if got := mapper.GetColor("test"); got != "blue" {
		t.Errorf("GetColor(test) on nil map = %q, want %q", got, "blue")
	}
}

func TestGet_EmptyMarker(t *testing.T) {
	mapper := NewMapper(map[string]string{
		"":    "empty-icon",
		".git": "",
	}, nil, nil, nil)

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
			}, nil, nil, nil)
			got := mapper.Get(tt.marker)
			if got != tt.icon {
				t.Errorf("Get(%q) = %q, want %q", tt.marker, got, tt.icon)
			}
		})
	}
}

func TestGetColor(t *testing.T) {
	tests := []struct {
		name     string
		colorMap map[string]string
		marker   string
		expected string
	}{
		{
			name:     "existing color",
			colorMap: map[string]string{".git": "red"},
			marker:   ".git",
			expected: "red",
		},
		{
			name:     "non-existent marker defaults to blue",
			colorMap: map[string]string{".git": "red"},
			marker:   "unknown",
			expected: "blue",
		},
		{
			name:     "empty map defaults to blue",
			colorMap: map[string]string{},
			marker:   ".git",
			expected: "blue",
		},
		{
			name:     "nil map defaults to blue",
			colorMap: nil,
			marker:   ".git",
			expected: "blue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapper := NewMapper(nil, tt.colorMap, nil, nil)
			got := mapper.GetColor(tt.marker)
			if got != tt.expected {
				t.Errorf("GetColor(%q) = %q, want %q", tt.marker, got, tt.expected)
			}
		})
	}
}

func TestSetColor(t *testing.T) {
	mapper := NewMapper(nil, map[string]string{
		".git": "red",
	}, nil, nil)

	// Test updating existing
	mapper.SetColor(".git", "cyan")
	if got := mapper.GetColor(".git"); got != "cyan" {
		t.Errorf("After SetColor(.git, cyan), GetColor(.git) = %q, want %q", got, "cyan")
	}

	// Test adding new
	mapper.SetColor("go.mod", "green")
	if got := mapper.GetColor("go.mod"); got != "green" {
		t.Errorf("After SetColor(go.mod, green), GetColor(go.mod) = %q, want %q", got, "green")
	}
}

func TestFormat(t *testing.T) {
	mapper := NewMapper(
		map[string]string{".git": "", "go.mod": "󰟓"},
		map[string]string{".git": "red", "go.mod": "cyan"},
		nil, nil,
	)

	tests := []struct {
		name     string
		marker   string
		ansi     bool
		expected string
	}{
		{
			name:     "ansi false returns plain icon",
			marker:   ".git",
			ansi:     false,
			expected: "",
		},
		{
			name:     "ansi true wraps with red",
			marker:   ".git",
			ansi:     true,
			expected: "\033[31m\033[39m",
		},
		{
			name:     "ansi true wraps with cyan",
			marker:   "go.mod",
			ansi:     true,
			expected: "\033[36m󰟓\033[39m",
		},
		{
			name:     "unknown marker with ansi uses default blue",
			marker:   "unknown",
			ansi:     true,
			expected: "\033[34m\033[39m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapper.Format(tt.marker, tt.ansi)
			if got != tt.expected {
				t.Errorf("Format(%q, %v) = %q, want %q", tt.marker, tt.ansi, got, tt.expected)
			}
		})
	}
}

func TestFormat_ANSICodes(t *testing.T) {
	tests := []struct {
		name      string
		color     string
		wantCode  string
	}{
		{name: "black", color: "black", wantCode: "\033[30m"},
		{name: "red", color: "red", wantCode: "\033[31m"},
		{name: "green", color: "green", wantCode: "\033[32m"},
		{name: "yellow", color: "yellow", wantCode: "\033[33m"},
		{name: "blue", color: "blue", wantCode: "\033[34m"},
		{name: "magenta", color: "magenta", wantCode: "\033[35m"},
		{name: "cyan", color: "cyan", wantCode: "\033[36m"},
		{name: "white", color: "white", wantCode: "\033[37m"},
		{name: "bright-black", color: "bright-black", wantCode: "\033[90m"},
		{name: "bright-red", color: "bright-red", wantCode: "\033[91m"},
		{name: "bright-green", color: "bright-green", wantCode: "\033[92m"},
		{name: "bright-yellow", color: "bright-yellow", wantCode: "\033[93m"},
		{name: "bright-blue", color: "bright-blue", wantCode: "\033[94m"},
		{name: "bright-magenta", color: "bright-magenta", wantCode: "\033[95m"},
		{name: "bright-cyan", color: "bright-cyan", wantCode: "\033[96m"},
		{name: "bright-white", color: "bright-white", wantCode: "\033[97m"},
		{name: "invalid color falls back to blue", color: "invalid", wantCode: "\033[34m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapper := NewMapper(
				map[string]string{"test": "X"},
				map[string]string{"test": tt.color},
				nil, nil,
			)
			got := mapper.Format("test", true)
			want := tt.wantCode + "X\033[39m"
			if got != want {
				t.Errorf("Format with color %q = %q, want %q", tt.color, got, want)
			}
		})
	}
}

func TestGetLabel(t *testing.T) {
	tests := []struct {
		name     string
		labelMap map[string]string
		marker   string
		expected string
	}{
		{
			name:     "existing label",
			labelMap: map[string]string{"go.mod": "go"},
			marker:   "go.mod",
			expected: "go",
		},
		{
			name:     "non-existent marker falls back to marker",
			labelMap: map[string]string{"go.mod": "go"},
			marker:   "unknown",
			expected: "unknown",
		},
		{
			name:     "nil map falls back to marker",
			labelMap: nil,
			marker:   ".git",
			expected: ".git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapper := NewMapper(nil, nil, tt.labelMap, nil)
			got := mapper.GetLabel(tt.marker)
			if got != tt.expected {
				t.Errorf("GetLabel(%q) = %q, want %q", tt.marker, got, tt.expected)
			}
		})
	}
}

func TestGetDisplayLabel(t *testing.T) {
	tests := []struct {
		name            string
		displayLabelMap map[string]string
		marker          string
		expected        string
	}{
		{
			name:            "existing display label",
			displayLabelMap: map[string]string{"go.mod": "Go"},
			marker:          "go.mod",
			expected:        "Go",
		},
		{
			name:            "non-existent marker returns empty",
			displayLabelMap: map[string]string{"go.mod": "Go"},
			marker:          "unknown",
			expected:        "",
		},
		{
			name:            "nil map returns empty",
			displayLabelMap: nil,
			marker:          ".git",
			expected:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapper := NewMapper(nil, nil, nil, tt.displayLabelMap)
			got := mapper.GetDisplayLabel(tt.marker)
			if got != tt.expected {
				t.Errorf("GetDisplayLabel(%q) = %q, want %q", tt.marker, got, tt.expected)
			}
		})
	}
}

func TestFormatLabel(t *testing.T) {
	tests := []struct {
		name     string
		label    string
		ansi     bool
		expected string
	}{
		{
			name:     "no ansi",
			label:    "go",
			ansi:     false,
			expected: "go",
		},
		{
			name:     "with ansi",
			label:    "go",
			ansi:     true,
			expected: "\033[2mgo\033[22m",
		},
		{
			name:     "empty label no ansi",
			label:    "",
			ansi:     false,
			expected: "",
		},
		{
			name:     "empty label with ansi",
			label:    "",
			ansi:     true,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatLabel(tt.label, tt.ansi)
			if got != tt.expected {
				t.Errorf("FormatLabel(%q, %v) = %q, want %q", tt.label, tt.ansi, got, tt.expected)
			}
		})
	}
}
