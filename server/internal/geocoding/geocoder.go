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
}

func NewGeocoder(logger *logrus.Logger, cacheDir string) *Geocoder {
	// Create cache directory if it doesn't exist
	os.MkdirAll(cacheDir, 0755)

	g := &Geocoder{
		logger:   logger,
		cacheDir: cacheDir,
		cache:    make(map[string][]float64),
		client:   &http.Client{Timeout: 10 * time.Second},
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
