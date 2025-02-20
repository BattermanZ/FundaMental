package config

import (
	"fundamental/server/internal/database"
)

// City represents a city configuration for the spider scheduler
type City struct {
	Name      string
	Center    []float64
	ZoomLevel int
}

// GetCityNames returns all cities from metropolitan areas
func GetCityNames(db *database.Database) ([]string, error) {
	areas, err := db.GetMetropolitanAreas()
	if err != nil {
		return nil, err
	}

	uniqueCities := make(map[string]bool)
	for _, area := range areas {
		for _, city := range area.Cities {
			uniqueCities[city] = true
		}
	}

	cities := make([]string, 0, len(uniqueCities))
	for city := range uniqueCities {
		cities = append(cities, city)
	}
	return cities, nil
}

// GetCityConfig returns configuration for a specific city
func GetCityConfig(db *database.Database, cityName string) (*City, error) {
	// Try to find the city in metropolitan areas
	areas, err := db.GetMetropolitanAreas()
	if err != nil {
		return nil, err
	}

	for _, area := range areas {
		for _, city := range area.Cities {
			if city == cityName {
				// Use metropolitan area configuration if available
				if area.CenterLat != nil && area.CenterLng != nil {
					return &City{
						Name:      city,
						Center:    []float64{*area.CenterLat, *area.CenterLng},
						ZoomLevel: getZoomLevel(area.ZoomLevel),
					}, nil
				}
			}
		}
	}

	// Fallback to default configuration
	return &City{
		Name:      cityName,
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
