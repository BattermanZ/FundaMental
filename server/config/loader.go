package config

import (
	"encoding/json"
	"fmt"
	"fundamental/server/internal/models"
	"os"
	"path/filepath"
	"sync"
)

var (
	metroConfig *models.MetropolitanConfig
	metroLock   sync.RWMutex
	metroPath   = "config/metropolitan_areas.json"
)

// LoadMetroConfig loads the metropolitan areas configuration from file
func LoadMetroConfig() error {
	metroLock.Lock()
	defer metroLock.Unlock()

	// Get absolute path to config file
	absPath, err := filepath.Abs(metroPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}

	// Read configuration file
	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	// Parse configuration
	var config models.MetropolitanConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %v", err)
	}

	metroConfig = &config
	return nil
}

// SaveMetroConfig saves the current configuration to file
func SaveMetroConfig() error {
	metroLock.Lock()
	defer metroLock.Unlock()

	if metroConfig == nil {
		return fmt.Errorf("no configuration loaded")
	}

	// Get absolute path to config file
	absPath, err := filepath.Abs(metroPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}

	// Marshal configuration with pretty printing
	data, err := json.MarshalIndent(metroConfig, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	// Write to file
	if err := os.WriteFile(absPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// GetMetroAreas returns all configured metropolitan areas
func GetMetroAreas() []models.MetropolitanArea {
	metroLock.RLock()
	defer metroLock.RUnlock()

	if metroConfig == nil {
		return nil
	}

	areas := make([]models.MetropolitanArea, len(metroConfig.MetropolitanAreas))
	for i, area := range metroConfig.MetropolitanAreas {
		areas[i] = models.MetropolitanArea{
			Name:   area.Name,
			Cities: area.Cities,
		}
	}
	return areas
}

// GetMetroAreaByName returns a specific metropolitan area by name
func GetMetroAreaByName(name string) *models.MetropolitanArea {
	metroLock.RLock()
	defer metroLock.RUnlock()

	if metroConfig == nil {
		return nil
	}

	for _, area := range metroConfig.MetropolitanAreas {
		if area.Name == name {
			return &models.MetropolitanArea{
				Name:   area.Name,
				Cities: area.Cities,
			}
		}
	}
	return nil
}

// UpdateMetroArea updates or adds a metropolitan area configuration
func UpdateMetroArea(area models.MetropolitanArea) error {
	metroLock.Lock()
	defer metroLock.Unlock()

	if metroConfig == nil {
		metroConfig = &models.MetropolitanConfig{}
	}

	// Find and update existing area or add new one
	found := false
	for i, existing := range metroConfig.MetropolitanAreas {
		if existing.Name == area.Name {
			metroConfig.MetropolitanAreas[i].Name = area.Name
			metroConfig.MetropolitanAreas[i].Cities = area.Cities
			found = true
			break
		}
	}

	if !found {
		metroConfig.MetropolitanAreas = append(metroConfig.MetropolitanAreas, struct {
			Name   string   `json:"name"`
			Cities []string `json:"cities"`
		}{
			Name:   area.Name,
			Cities: area.Cities,
		})
	}

	return SaveMetroConfig()
}

// DeleteMetroArea removes a metropolitan area configuration
func DeleteMetroArea(name string) error {
	metroLock.Lock()
	defer metroLock.Unlock()

	if metroConfig == nil {
		return fmt.Errorf("no configuration loaded")
	}

	// Find and remove the area
	for i, area := range metroConfig.MetropolitanAreas {
		if area.Name == name {
			metroConfig.MetropolitanAreas = append(
				metroConfig.MetropolitanAreas[:i],
				metroConfig.MetropolitanAreas[i+1:]...,
			)
			return SaveMetroConfig()
		}
	}

	return fmt.Errorf("metropolitan area not found: %s", name)
}

// GetCitiesInMetro returns all cities in a metropolitan area
func GetCitiesInMetro(name string) []string {
	area := GetMetroAreaByName(name)
	if area == nil {
		return nil
	}
	return area.Cities
}
