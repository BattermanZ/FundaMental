package config

import (
	"fundamental/server/internal/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDatabase is a mock implementation of the DatabaseReader interface
type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) GetMetropolitanAreas() ([]models.MetropolitanArea, error) {
	args := m.Called()
	return args.Get(0).([]models.MetropolitanArea), args.Error(1)
}

func TestGetCityNames(t *testing.T) {
	tests := []struct {
		name           string
		areas          []models.MetropolitanArea
		expectedCities []string
		expectError    bool
	}{
		{
			name: "Basic city list",
			areas: []models.MetropolitanArea{
				{
					ID:     1,
					Name:   "Amsterdam Metro",
					Cities: []string{"Amsterdam", "Amstelveen", "Diemen"},
				},
				{
					ID:     2,
					Name:   "Rotterdam Metro",
					Cities: []string{"Rotterdam", "Schiedam", "Vlaardingen"},
				},
			},
			expectedCities: []string{
				"Amsterdam", "Amstelveen", "Diemen",
				"Rotterdam", "Schiedam", "Vlaardingen",
			},
			expectError: false,
		},
		{
			name: "Duplicate cities",
			areas: []models.MetropolitanArea{
				{
					ID:     1,
					Name:   "Area 1",
					Cities: []string{"Amsterdam", "Diemen"},
				},
				{
					ID:     2,
					Name:   "Area 2",
					Cities: []string{"Amsterdam", "Amstelveen"},
				},
			},
			expectedCities: []string{"Amsterdam", "Diemen", "Amstelveen"},
			expectError:    false,
		},
		{
			name: "Empty areas",
			areas: []models.MetropolitanArea{
				{
					ID:     1,
					Name:   "Empty Area",
					Cities: []string{},
				},
			},
			expectedCities: []string{},
			expectError:    false,
		},
		{
			name: "Cities with special characters",
			areas: []models.MetropolitanArea{
				{
					ID:     1,
					Name:   "Special Area",
					Cities: []string{"'s-Hertogenbosch", "Den Haag", "Delft"},
				},
			},
			expectedCities: []string{"'s-Hertogenbosch", "Den Haag", "Delft"},
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDatabase{}
			mockDB.On("GetMetropolitanAreas").Return(tt.areas, nil)

			cities, err := GetCityNames(mockDB)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.ElementsMatch(t, tt.expectedCities, cities,
					"Cities should match regardless of order")
			}

			mockDB.AssertExpectations(t)
		})
	}
}

func TestGetCityConfig(t *testing.T) {
	tests := []struct {
		name           string
		areas          []models.MetropolitanArea
		cityName       string
		expectedConfig *City
		expectError    bool
	}{
		{
			name: "City with coordinates",
			areas: []models.MetropolitanArea{
				{
					ID:        1,
					Name:      "Amsterdam Metro",
					Cities:    []string{"Amsterdam"},
					CenterLat: ptr(52.3676),
					CenterLng: ptr(4.9041),
					ZoomLevel: ptrInt(13),
				},
			},
			cityName: "Amsterdam",
			expectedConfig: &City{
				Name:      "Amsterdam",
				Center:    []float64{52.3676, 4.9041},
				ZoomLevel: 13,
			},
			expectError: false,
		},
		{
			name: "City without coordinates",
			areas: []models.MetropolitanArea{
				{
					ID:     1,
					Name:   "Rotterdam Metro",
					Cities: []string{"Rotterdam"},
				},
			},
			cityName: "Rotterdam",
			expectedConfig: &City{
				Name:      "Rotterdam",
				Center:    []float64{52.3676, 4.9041}, // Default Amsterdam coordinates
				ZoomLevel: 13,
			},
			expectError: false,
		},
		{
			name: "Non-existent city",
			areas: []models.MetropolitanArea{
				{
					ID:     1,
					Name:   "Amsterdam Metro",
					Cities: []string{"Amsterdam"},
				},
			},
			cityName: "Utrecht",
			expectedConfig: &City{
				Name:      "Utrecht",
				Center:    []float64{52.3676, 4.9041}, // Default Amsterdam coordinates
				ZoomLevel: 13,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDatabase{}
			mockDB.On("GetMetropolitanAreas").Return(tt.areas, nil)

			config, err := GetCityConfig(mockDB, tt.cityName)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedConfig.Name, config.Name)
				assert.Equal(t, tt.expectedConfig.ZoomLevel, config.ZoomLevel)
				assert.InDelta(t, tt.expectedConfig.Center[0], config.Center[0], 0.0001)
				assert.InDelta(t, tt.expectedConfig.Center[1], config.Center[1], 0.0001)
			}

			mockDB.AssertExpectations(t)
		})
	}
}

func TestNormalizeCity(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple city name",
			input:    "Amsterdam",
			expected: "amsterdam",
		},
		{
			name:     "City name with spaces",
			input:    "Den Haag",
			expected: "den-haag",
		},
		{
			name:     "City name with apostrophe",
			input:    "'s-Hertogenbosch",
			expected: "s-hertogenbosch",
		},
		{
			name:     "Mixed case with spaces",
			input:    "Alphen aan den Rijn",
			expected: "alphen-aan-den-rijn",
		},
		{
			name:     "Already normalized",
			input:    "utrecht",
			expected: "utrecht",
		},
		{
			name:     "Multiple spaces",
			input:    "Bergen  op  Zoom",
			expected: "bergen-op-zoom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeCity(tt.input)
			assert.Equal(t, tt.expected, result,
				"NormalizeCity(%q) = %q, want %q", tt.input, result, tt.expected)
		})
	}
}

// Helper function to create pointer to float64
func ptr(v float64) *float64 {
	return &v
}

// Helper function to create pointer to int
func ptrInt(v int) *int {
	return &v
}
