package icons

import "fmt"

// ANSI color codes for foreground colors
var ansiColors = map[string]int{
	"black":          30,
	"red":            31,
	"green":          32,
	"yellow":         33,
	"blue":           34,
	"magenta":        35,
	"cyan":           36,
	"white":          37,
	"bright-black":   90,
	"bright-red":     91,
	"bright-green":   92,
	"bright-yellow":  93,
	"bright-blue":    94,
	"bright-magenta": 95,
	"bright-cyan":    96,
	"bright-white":   97,
}

// Mapper handles icon, color, label, and display label mapping for project markers
type Mapper struct {
	iconMap         map[string]string
	colorMap        map[string]string
	labelMap        map[string]string
	displayLabelMap map[string]string
}

// NewMapper creates a new icon mapper with the given maps
func NewMapper(iconMap, colorMap, labelMap, displayLabelMap map[string]string) *Mapper {
	// Create copies to avoid mutations
	im := make(map[string]string)
	for k, v := range iconMap {
		im[k] = v
	}
	cm := make(map[string]string)
	for k, v := range colorMap {
		cm[k] = v
	}
	lm := make(map[string]string)
	for k, v := range labelMap {
		lm[k] = v
	}
	dm := make(map[string]string)
	for k, v := range displayLabelMap {
		dm[k] = v
	}
	return &Mapper{iconMap: im, colorMap: cm, labelMap: lm, displayLabelMap: dm}
}

// Get returns the icon for a given marker
func (m *Mapper) Get(marker string) string {
	if icon, ok := m.iconMap[marker]; ok {
		return icon
	}
	// Return a default icon if not found
	return ""
}

// Set updates or adds an icon mapping
func (m *Mapper) Set(marker, icon string) {
	m.iconMap[marker] = icon
}

// GetColor returns the color name for a given marker, defaulting to "blue"
func (m *Mapper) GetColor(marker string) string {
	if color, ok := m.colorMap[marker]; ok {
		return color
	}
	return "blue"
}

// SetColor updates or adds a color mapping
func (m *Mapper) SetColor(marker, color string) {
	m.colorMap[marker] = color
}

// GetLabel returns the semantic label for a given marker, falling back to the marker itself
func (m *Mapper) GetLabel(marker string) string {
	if label, ok := m.labelMap[marker]; ok {
		return label
	}
	return marker
}

// GetDisplayLabel returns the human-readable display label for a given marker, or empty string if not set
func (m *Mapper) GetDisplayLabel(marker string) string {
	if label, ok := m.displayLabelMap[marker]; ok {
		return label
	}
	return ""
}

// Format returns the icon for a marker, optionally wrapped in ANSI color codes.
// When ansi is true, the icon is wrapped as \033[<code>m<icon>\033[39m.
// When ansi is false, the plain icon is returned.
func (m *Mapper) Format(marker string, ansi bool) string {
	icon := m.Get(marker)
	if !ansi {
		return icon
	}
	color := m.GetColor(marker)
	code, ok := ansiColors[color]
	if !ok {
		code = ansiColors["blue"]
	}
	return fmt.Sprintf("\033[%dm%s\033[39m", code, icon)
}

func FormatLabel(label string, ansi bool) string {
	if !ansi || label == "" {
		return label
	}
	return fmt.Sprintf("\033[2m%s\033[22m", label)
}
