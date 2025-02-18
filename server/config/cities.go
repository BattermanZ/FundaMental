package config

// City represents a city configuration
type City struct {
	Name      string    `json:"name"`
	Center    []float64 `json:"center"`
	ZoomLevel int       `json:"zoom_level"`
}

// SupportedCities is a list of cities supported by the application
var SupportedCities = []City{
	{
		Name:      "amsterdam",
		Center:    []float64{52.3676, 4.9041},
		ZoomLevel: 13,
	},
	// Add more cities here as needed
}

// GetCityNames returns a list of supported city names
func GetCityNames() []string {
	names := make([]string, len(SupportedCities))
	for i, city := range SupportedCities {
		names[i] = city.Name
	}
	return names
}

// GetCityByName returns a city configuration by name
func GetCityByName(name string) *City {
	for _, city := range SupportedCities {
		if city.Name == name {
			return &city
		}
	}
	return nil
}
