package icons

// Mapper handles icon mapping for project markers
type Mapper struct {
	iconMap map[string]string
}

// NewMapper creates a new icon mapper with the given icon map
func NewMapper(iconMap map[string]string) *Mapper {
	// Create a copy to avoid mutations
	m := make(map[string]string)
	for k, v := range iconMap {
		m[k] = v
	}
	return &Mapper{iconMap: m}
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
