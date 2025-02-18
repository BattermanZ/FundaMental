package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// MetropolitanArea represents a metropolitan area configuration
type MetropolitanArea struct {
	Name   string   `json:"name"`
	Cities []string `json:"cities"`
}

// MetropolitanConfig represents the full metropolitan areas configuration
type MetropolitanConfig struct {
	MetropolitanAreas []MetropolitanArea `json:"metropolitan_areas"`
}

var (
	metropolitanConfig *MetropolitanConfig
	configLock         sync.RWMutex
	configPath         = "config/metropolitan_areas.json"
)

// LoadMetropolitanConfig loads the metropolitan areas configuration from file
func LoadMetropolitanConfig() error {
	configLock.Lock()
	defer configLock.Unlock()

	// Get absolute path to config file
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}

	// Read configuration file
	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	// Parse configuration
	var config MetropolitanConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %v", err)
	}

	metropolitanConfig = &config
	return nil
}

// SaveMetropolitanConfig saves the current configuration to file
func SaveMetropolitanConfig() error {
	configLock.Lock()
	defer configLock.Unlock()

	if metropolitanConfig == nil {
		return fmt.Errorf("no configuration loaded")
	}

	// Get absolute path to config file
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}

	// Marshal configuration with pretty printing
	data, err := json.MarshalIndent(metropolitanConfig, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	// Write to file
	if err := os.WriteFile(absPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// GetMetropolitanAreas returns all configured metropolitan areas
func GetMetropolitanAreas() []MetropolitanArea {
	configLock.RLock()
	defer configLock.RUnlock()

	if metropolitanConfig == nil {
		return nil
	}
	return metropolitanConfig.MetropolitanAreas
}

// GetMetropolitanAreaByName returns a specific metropolitan area by name
func GetMetropolitanAreaByName(name string) *MetropolitanArea {
	configLock.RLock()
	defer configLock.RUnlock()

	if metropolitanConfig == nil {
		return nil
	}

	for _, area := range metropolitanConfig.MetropolitanAreas {
		if area.Name == name {
			return &area
		}
	}
	return nil
}

// UpdateMetropolitanArea updates or adds a metropolitan area configuration
func UpdateMetropolitanArea(area MetropolitanArea) error {
	configLock.Lock()
	defer configLock.Unlock()

	if metropolitanConfig == nil {
		metropolitanConfig = &MetropolitanConfig{}
	}

	// Find and update existing area or add new one
	found := false
	for i, existing := range metropolitanConfig.MetropolitanAreas {
		if existing.Name == area.Name {
			metropolitanConfig.MetropolitanAreas[i] = area
			found = true
			break
		}
	}

	if !found {
		metropolitanConfig.MetropolitanAreas = append(metropolitanConfig.MetropolitanAreas, area)
	}

	return SaveMetropolitanConfig()
}

// DeleteMetropolitanArea removes a metropolitan area configuration
func DeleteMetropolitanArea(name string) error {
	configLock.Lock()
	defer configLock.Unlock()

	if metropolitanConfig == nil {
		return fmt.Errorf("no configuration loaded")
	}

	// Find and remove the area
	for i, area := range metropolitanConfig.MetropolitanAreas {
		if area.Name == name {
			metropolitanConfig.MetropolitanAreas = append(
				metropolitanConfig.MetropolitanAreas[:i],
				metropolitanConfig.MetropolitanAreas[i+1:]...,
			)
			return SaveMetropolitanConfig()
		}
	}

	return fmt.Errorf("metropolitan area not found: %s", name)
}

// GetCitiesInMetropolitanArea returns all cities in a metropolitan area
func GetCitiesInMetropolitanArea(name string) []string {
	area := GetMetropolitanAreaByName(name)
	if area == nil {
		return nil
	}
	return area.Cities
}
