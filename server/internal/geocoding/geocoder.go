package geocoding

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type Geocoder struct {
	logger    *logrus.Logger
	cacheDir  string
	cache     map[string][]float64
	cacheLock sync.RWMutex
	client    *http.Client
	rateLimit time.Duration
	lastCall  time.Time
}

type GeocodingResult struct {
	Lat float64
	Lng float64
}

const (
	// Netherlands bounding box
	NL_MIN_LAT = 50.75
	NL_MAX_LAT = 53.55
	NL_MIN_LNG = 3.35
	NL_MAX_LNG = 7.22
)

func NewGeocoder(logger *logrus.Logger, cacheDir string) *Geocoder {
	// Create cache directory if it doesn't exist
	os.MkdirAll(cacheDir, 0755)

	g := &Geocoder{
		logger:    logger,
		cacheDir:  cacheDir,
		cache:     make(map[string][]float64),
		client:    &http.Client{Timeout: 10 * time.Second},
		rateLimit: time.Second, // 1 request per second
	}

	// Load cache from file
	g.loadCache()

	return g
}

func (g *Geocoder) loadCache() {
	cacheFile := filepath.Join(g.cacheDir, "geocode_cache.json")
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		g.logger.Warnf("Could not load geocode cache: %v", err)
		return
	}

	err = json.Unmarshal(data, &g.cache)
	if err != nil {
		g.logger.Errorf("Failed to parse geocode cache: %v", err)
		return
	}

	g.logger.Infof("Loaded %d cached addresses", len(g.cache))
}

func (g *Geocoder) saveCache() {
	g.cacheLock.RLock()
	defer g.cacheLock.RUnlock()

	cacheFile := filepath.Join(g.cacheDir, "geocode_cache.json")
	data, err := json.Marshal(g.cache)
	if err != nil {
		g.logger.Errorf("Failed to marshal geocode cache: %v", err)
		return
	}

	err = os.WriteFile(cacheFile, data, 0644)
	if err != nil {
		g.logger.Errorf("Failed to save geocode cache: %v", err)
		return
	}

	g.logger.Info("Saved geocode cache to disk")
}

type nominatimResponse []struct {
	Lat string `json:"lat"`
	Lon string `json:"lon"`
}

func (g *Geocoder) GeocodeAddress(street, postalCode, city string) (float64, float64, error) {
	cacheKey := fmt.Sprintf("%s|%s|%s", street, postalCode, city)
	fullAddress := fmt.Sprintf("%s, %s, %s, Netherlands", street, postalCode, city)

	// Check cache first
	g.cacheLock.RLock()
	if coords, ok := g.cache[cacheKey]; ok {
		g.cacheLock.RUnlock()
		if len(coords) == 2 {
			g.logger.WithFields(logrus.Fields{
				"address":   fullAddress,
				"latitude":  coords[0],
				"longitude": coords[1],
				"source":    "cache",
			}).Info("Found coordinates in cache")
			return coords[0], coords[1], nil
		}
		return 0, 0, fmt.Errorf("invalid cached coordinates")
	}
	g.cacheLock.RUnlock()

	g.logger.WithField("address", fullAddress).Info("Geocoding address with Nominatim")

	// Respect Nominatim's usage policy
	time.Sleep(time.Second)

	// Build the query
	params := url.Values{
		"q":              []string{fullAddress},
		"format":         []string{"json"},
		"limit":          []string{"1"},
		"countrycodes":   []string{"nl"},
		"addressdetails": []string{"1"},
	}

	// Make the request
	req, err := http.NewRequest("GET", "https://nominatim.openstreetmap.org/search", nil)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create request: %v", err)
	}

	req.URL.RawQuery = params.Encode()
	req.Header.Set("User-Agent", "FundaMental Property Analyzer/1.0")
	req.Header.Set("Accept-Language", "nl-NL,nl;q=0.9,en-US;q=0.8,en;q=0.7")

	resp, err := g.client.Do(req)
	if err != nil {
		g.logger.WithError(err).WithField("address", fullAddress).Error("Geocoding request failed")
		return 0, 0, fmt.Errorf("geocoding request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		g.logger.WithError(err).WithField("address", fullAddress).Error("Failed to read response")
		return 0, 0, fmt.Errorf("failed to read response: %v", err)
	}

	var result nominatimResponse
	if err := json.Unmarshal(body, &result); err != nil {
		g.logger.WithError(err).WithField("address", fullAddress).Error("Failed to parse response")
		return 0, 0, fmt.Errorf("failed to parse response: %v", err)
	}

	if len(result) == 0 {
		g.logger.WithField("address", fullAddress).Warn("No results found")
		return 0, 0, fmt.Errorf("no results found for address: %s", fullAddress)
	}

	var lat, lon float64
	fmt.Sscanf(result[0].Lat, "%f", &lat)
	fmt.Sscanf(result[0].Lon, "%f", &lon)

	g.logger.WithFields(logrus.Fields{
		"address":   fullAddress,
		"latitude":  lat,
		"longitude": lon,
		"source":    "nominatim",
	}).Info("Successfully geocoded address")

	// Cache the result
	g.cacheLock.Lock()
	g.cache[cacheKey] = []float64{lat, lon}
	g.cacheLock.Unlock()

	// Save cache periodically
	go g.saveCache()

	return lat, lon, nil
}

// GeocodeCity geocodes a city name with country context
func (g *Geocoder) GeocodeCity(city string) (*GeocodingResult, error) {
	// Check cache first
	if result := g.getCityFromCache(city); result != nil {
		g.logger.Infof("Found city %s in cache", city)
		return result, nil
	}

	// Rate limiting
	if time.Since(g.lastCall) < g.rateLimit {
		time.Sleep(g.rateLimit - time.Since(g.lastCall))
	}
	g.lastCall = time.Now()

	// Construct the query with Netherlands context
	query := fmt.Sprintf("%s, Netherlands", city)
	encodedQuery := url.QueryEscape(query)
	url := fmt.Sprintf("https://nominatim.openstreetmap.org/search?q=%s&format=json&limit=1", encodedQuery)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set User-Agent as required by Nominatim
	req.Header.Set("User-Agent", "FundaMental/1.0")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("geocoding request failed: %v", err)
	}
	defer resp.Body.Close()

	var results []struct {
		Lat string `json:"lat"`
		Lon string `json:"lon"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no results found for city: %s", city)
	}

	// Parse coordinates
	var lat, lng float64
	fmt.Sscanf(results[0].Lat, "%f", &lat)
	fmt.Sscanf(results[0].Lon, "%f", &lng)

	// Validate coordinates are within Netherlands
	if !g.isWithinNetherlands(lat, lng) {
		return nil, fmt.Errorf("coordinates for %s are outside Netherlands bounds", city)
	}

	result := &GeocodingResult{
		Lat: lat,
		Lng: lng,
	}

	// Cache the result
	g.cacheCityResult(city, result)

	return result, nil
}

func (g *Geocoder) isWithinNetherlands(lat, lng float64) bool {
	return lat >= NL_MIN_LAT && lat <= NL_MAX_LAT &&
		lng >= NL_MIN_LNG && lng <= NL_MAX_LNG
}

func (g *Geocoder) getCityFromCache(city string) *GeocodingResult {
	cacheFile := filepath.Join(g.cacheDir, fmt.Sprintf("city_%s.json", city))
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil
	}

	var result GeocodingResult
	if err := json.Unmarshal(data, &result); err != nil {
		g.logger.Warnf("Failed to unmarshal cached city data: %v", err)
		return nil
	}

	return &result
}

func (g *Geocoder) cacheCityResult(city string, result *GeocodingResult) {
	data, err := json.Marshal(result)
	if err != nil {
		g.logger.Warnf("Failed to marshal city result: %v", err)
		return
	}

	cacheFile := filepath.Join(g.cacheDir, fmt.Sprintf("city_%s.json", city))
	if err := os.MkdirAll(g.cacheDir, 0755); err != nil {
		g.logger.Warnf("Failed to create cache directory: %v", err)
		return
	}

	if err := os.WriteFile(cacheFile, data, 0644); err != nil {
		g.logger.Warnf("Failed to write city cache file: %v", err)
	}
}
