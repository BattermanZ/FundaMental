package models

import "time"

type Property struct {
	ID           int64     `json:"id"`
	URL          string    `json:"url"`
	Street       string    `json:"street"`
	Neighborhood string    `json:"neighborhood"`
	PropertyType string    `json:"property_type"`
	City         string    `json:"city"`
	PostalCode   string    `json:"postal_code"`
	Price        int       `json:"price"`
	YearBuilt    *int      `json:"year_built"`
	LivingArea   *int      `json:"living_area"`
	NumRooms     *int      `json:"num_rooms"`
	Status       string    `json:"status"`
	ListingDate  time.Time `json:"listing_date"`
	SellingDate  time.Time `json:"selling_date"`
	ScrapedAt    time.Time `json:"scraped_at"`
	CreatedAt    time.Time `json:"created_at"`
	Latitude     *float64  `json:"latitude"`
	Longitude    *float64  `json:"longitude"`
	EnergyLabel  string    `json:"energy_label"`
}

type PropertyStats struct {
	TotalProperties int     `json:"total_properties"`
	AveragePrice    float64 `json:"average_price"`
	MedianPrice     float64 `json:"median_price"`
	AvgDaysToSell   float64 `json:"avg_days_to_sell"`
	TotalSold       int     `json:"total_sold"`
	TotalActive     int     `json:"total_active"`
	PricePerSqm     float64 `json:"price_per_sqm"`
}

type AreaStats struct {
	PostalCode     string  `json:"postal_code"`
	PropertyCount  int     `json:"property_count"`
	AveragePrice   float64 `json:"average_price"`
	MedianPrice    float64 `json:"median_price"`
	AvgPricePerSqm float64 `json:"avg_price_per_sqm"`
}

type MetropolitanArea struct {
	ID        int64    `json:"id"`
	Name      string   `json:"name"`
	Cities    []string `json:"cities"`
	CenterLat *float64 `json:"center_lat,omitempty"`
	CenterLng *float64 `json:"center_lng,omitempty"`
	ZoomLevel *int     `json:"zoom_level,omitempty"`
}

type MetropolitanCity struct {
	ID                 int64   `json:"id"`
	MetropolitanAreaID int64   `json:"metropolitan_area_id"`
	City               string  `json:"city"`
	Lat                float64 `json:"lat,omitempty"`
	Lng                float64 `json:"lng,omitempty"`
}

// MetropolitanConfig represents the configuration format for metropolitan areas
type MetropolitanConfig struct {
	MetropolitanAreas []struct {
		Name   string   `json:"name"`
		Cities []string `json:"cities"`
	} `json:"metropolitan_areas"`
}
