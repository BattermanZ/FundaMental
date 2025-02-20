package config

import (
	"fundamental/server/internal/models"
	"regexp"
	"strings"
)

// DatabaseReader interface defines the methods needed for city configuration
type DatabaseReader interface {
	GetMetropolitanAreas() ([]models.MetropolitanArea, error)
}

// City represents a city configuration for the spider scheduler
type City struct {
	Name      string
	Center    []float64
	ZoomLevel int
}

var multipleSpacesRegex = regexp.MustCompile(`\s+`)

// NormalizeCity ensures consistent city name formatting for Funda
func NormalizeCity(city string) string {
	// Convert to lowercase for comparison
	normalized := strings.ToLower(city)

	// Special case: 's-Hertogenbosch is referenced as den-bosch on Funda
	if normalized == "'s-hertogenbosch" || normalized == "s-hertogenbosch" {
		return "den-bosch"
	}

	// Replace multiple spaces with a single space
	normalized = multipleSpacesRegex.ReplaceAllString(normalized, " ")

	// Replace spaces with hyphens
	normalized = strings.ReplaceAll(normalized, " ", "-")

	// Remove any apostrophes
	normalized = strings.ReplaceAll(normalized, "'", "")

	return normalized
}

// GetCityNames returns all cities from metropolitan areas
func GetCityNames(db DatabaseReader) ([]string, error) {
	areas, err := db.GetMetropolitanAreas()
	if err != nil {
		return nil, err
	}

	uniqueCities := make(map[string]string) // map[normalized]original
	for _, area := range areas {
		for _, city := range area.Cities {
			normalized := NormalizeCity(city)
			uniqueCities[normalized] = city
		}
	}

	cities := make([]string, 0, len(uniqueCities))
	for _, city := range uniqueCities {
		cities = append(cities, city)
	}
	return cities, nil
}

// GetCityConfig returns configuration for a specific city
func GetCityConfig(db DatabaseReader, cityName string) (*City, error) {
	// Normalize the input city name for comparison
	normalizedInput := NormalizeCity(cityName)

	// Try to find the city in metropolitan areas
	areas, err := db.GetMetropolitanAreas()
	if err != nil {
		return nil, err
	}

	for _, area := range areas {
		for _, city := range area.Cities {
			if NormalizeCity(city) == normalizedInput {
				// Use metropolitan area configuration if available
				if area.CenterLat != nil && area.CenterLng != nil {
					return &City{
						Name:      city, // Use original city name
						Center:    []float64{*area.CenterLat, *area.CenterLng},
						ZoomLevel: getZoomLevel(area.ZoomLevel),
					}, nil
				}
			}
		}
	}

	// Fallback to default configuration
	return &City{
		Name:      cityName, // Use original input name
		Center:    getDefaultCenter(cityName),
		ZoomLevel: 13,
	}, nil
}

// Helper function to get zoom level with fallback
func getZoomLevel(z *int) int {
	if z != nil {
		return *z
	}
	return 13 // Default zoom level
}

// Helper function to get default center coordinates
func getDefaultCenter(cityName string) []float64 {
	// Could be expanded to include default coordinates for major cities
	return []float64{52.3676, 4.9041} // Amsterdam coordinates as default
}
