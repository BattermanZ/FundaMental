package geometry

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/sirupsen/logrus"
)

type DistrictPoint struct {
	Latitude  float64
	Longitude float64
}

type District struct {
	Code   string
	City   string
	Points []DistrictPoint
	Hull   *geojson.Feature
}

type DistrictManager struct {
	db     *sql.DB
	logger *logrus.Logger
}

type PDOKResponse struct {
	Response struct {
		Docs []struct {
			CentroidLL string `json:"centroide_ll"`
		} `json:"docs"`
	} `json:"response"`
}

func NewDistrictManager(db *sql.DB, logger *logrus.Logger) *DistrictManager {
	return &DistrictManager{
		db:     db,
		logger: logger,
	}
}

func (dm *DistrictManager) CleanPreviousData() error {
	outputPath := filepath.Join("..", "client", "public", "district_hulls.geojson")
	if err := os.Remove(outputPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove previous data: %v", err)
	}
	return nil
}

func (dm *DistrictManager) GetUniqueDistricts() (map[string]string, error) {
	// Query to get unique postal districts (first 4 digits) and their cities
	query := `
		SELECT DISTINCT 
			substr(postal_code, 1, 4) as district,
			city
		FROM properties 
		WHERE postal_code IS NOT NULL
		  AND length(postal_code) >= 4
		  AND postal_code GLOB '[0-9][0-9][0-9][0-9]*'  -- Ensure valid postal code format (4 digits)
	`

	rows, err := dm.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query districts: %v", err)
	}
	defer rows.Close()

	districts := make(map[string]string) // map[district]city

	for rows.Next() {
		var district, city string
		if err := rows.Scan(&district, &city); err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}
		districts[district] = city
	}

	return districts, nil
}

func (dm *DistrictManager) FetchDistrictPoints(district string, city string) ([]DistrictPoint, error) {
	baseURL := "https://api.pdok.nl/bzk/locatieserver/search/v3_1/free"

	// Build query parameters
	params := url.Values{}
	params.Set("q", fmt.Sprintf("type:postcode AND postcode:%s* AND woonplaatsnaam:%s", district, city))
	params.Set("rows", "100")
	params.Set("fl", "*")
	params.Set("fq", "type:postcode")

	// Create request
	req, err := http.NewRequest("GET", baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Add headers
	req.Header.Set("User-Agent", "FundaMental Property Analyzer/1.0")
	req.Header.Set("Accept-Language", "nl-NL,nl;q=0.9,en-US;q=0.8,en;q=0.7")

	// Make request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Parse response
	var pdokResp PDOKResponse
	if err := json.Unmarshal(body, &pdokResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	// Extract points
	var points []DistrictPoint
	seen := make(map[string]bool) // To deduplicate points

	for _, doc := range pdokResp.Response.Docs {
		var lat, lon float64
		_, err := fmt.Sscanf(doc.CentroidLL, "POINT(%f %f)", &lon, &lat)
		if err != nil {
			dm.logger.Warnf("Failed to parse coordinates from %s: %v", doc.CentroidLL, err)
			continue
		}

		// Deduplicate points
		key := fmt.Sprintf("%.6f,%.6f", lat, lon)
		if !seen[key] {
			points = append(points, DistrictPoint{
				Latitude:  lat,
				Longitude: lon,
			})
			seen[key] = true
		}
	}

	// Add delay to respect rate limits
	time.Sleep(100 * time.Millisecond)

	return points, nil
}

func angle(center, p orb.Point) float64 {
	dx := p[0] - center[0]
	dy := p[1] - center[1]
	return -1 * float64(int64(dy)) / (dx*dx + dy*dy + 1e-10)
}

func sortPointsByAngle(points []orb.Point, center orb.Point) {
	sort.Slice(points, func(i, j int) bool {
		angleI := angle(center, points[i])
		angleJ := angle(center, points[j])
		return angleI < angleJ
	})
}

func distance(p1, p2 orb.Point) float64 {
	dx := p2[0] - p1[0]
	dy := p2[1] - p1[1]
	return dx*dx + dy*dy
}

func interpolatePoints(p1, p2 orb.Point, t float64) orb.Point {
	return orb.Point{
		p1[0] + t*(p2[0]-p1[0]),
		p1[1] + t*(p2[1]-p1[1]),
	}
}

func bufferHull(hull orb.Ring, bufferDistance float64) orb.Ring {
	if len(hull) < 4 {
		return hull
	}

	// Create a new ring with interpolated points
	var buffered []orb.Point
	numPoints := len(hull)

	for i := 0; i < numPoints-1; i++ {
		p1 := hull[i]
		p2 := hull[(i+1)%numPoints]

		// Add original point
		buffered = append(buffered, p1)

		// Calculate distance between points
		dist := distance(p1, p2)
		if dist > bufferDistance*bufferDistance*4 {
			// Add interpolated points if points are far apart
			numInterpolated := int(dist / (bufferDistance * bufferDistance))
			for j := 1; j < numInterpolated; j++ {
				t := float64(j) / float64(numInterpolated)
				buffered = append(buffered, interpolatePoints(p1, p2, t))
			}
		}
	}

	// Close the ring
	buffered = append(buffered, buffered[0])

	// Smooth the corners
	smoothed := make([]orb.Point, 0, len(buffered))
	for i := 0; i < len(buffered)-1; i++ {
		p1 := buffered[i]
		p2 := buffered[(i+1)%len(buffered)]
		p3 := buffered[(i+2)%len(buffered)]

		// Add the current point
		smoothed = append(smoothed, p1)

		// Add rounded corner points
		v1x := p2[0] - p1[0]
		v1y := p2[1] - p1[1]
		v2x := p3[0] - p2[0]
		v2y := p3[1] - p2[1]

		// Normalize vectors
		v1len := distance(p1, p2)
		v2len := distance(p2, p3)
		if v1len > 0 && v2len > 0 {
			v1x /= v1len
			v1y /= v1len
			v2x /= v2len
			v2y /= v2len

			// Calculate angle between vectors
			dot := v1x*v2x + v1y*v2y
			if dot < 0.9 { // Only round sharp corners
				// Add arc points
				numArcPoints := 5
				for j := 1; j < numArcPoints; j++ {
					t := float64(j) / float64(numArcPoints)
					smoothed = append(smoothed, orb.Point{
						p2[0] + bufferDistance*(-v1x*t+v2x*(1-t)),
						p2[1] + bufferDistance*(-v1y*t+v2y*(1-t)),
					})
				}
			}
		}
	}

	// Close the ring
	smoothed = append(smoothed, smoothed[0])

	return orb.Ring(smoothed)
}

func generateConvexHull(points []orb.Point) orb.Ring {
	if len(points) < 3 {
		return nil
	}

	// Find the leftmost point
	leftmost := points[0]
	leftmostIdx := 0
	for i := 1; i < len(points); i++ {
		if points[i][0] < leftmost[0] {
			leftmost = points[i]
			leftmostIdx = i
		}
	}

	// Move leftmost point to first position
	points[0], points[leftmostIdx] = points[leftmostIdx], points[0]

	// Sort remaining points by angle
	sortPointsByAngle(points[1:], points[0])

	// Graham scan
	hull := []orb.Point{points[0], points[1]}
	for i := 2; i < len(points); i++ {
		for len(hull) > 1 {
			n := len(hull)
			// Calculate cross product
			v1x := hull[n-1][0] - hull[n-2][0]
			v1y := hull[n-1][1] - hull[n-2][1]
			v2x := points[i][0] - hull[n-2][0]
			v2y := points[i][1] - hull[n-2][1]
			cross := v1x*v2y - v1y*v2x

			if cross >= 0 {
				break
			}
			hull = hull[:n-1]
		}
		hull = append(hull, points[i])
	}

	// Close the ring
	if len(hull) > 2 {
		hull = append(hull, hull[0])
	}

	// Buffer the hull to create smoother boundaries
	return bufferHull(orb.Ring(hull), 0.001)
}

func (dm *DistrictManager) GenerateHulls(districts map[string]*District) error {
	for key, district := range districts {
		if len(district.Points) < 3 {
			dm.logger.Warnf("Not enough points for district %s (minimum 3 required)", key)
			continue
		}

		// Convert points to orb.Point slice
		points := make([]orb.Point, len(district.Points))
		for i, p := range district.Points {
			points[i] = orb.Point{p.Longitude, p.Latitude}
		}

		// Generate convex hull
		hull := generateConvexHull(points)
		if hull == nil {
			continue
		}

		// Create GeoJSON feature
		feature := geojson.NewFeature(hull)
		feature.Properties = geojson.Properties{
			"district":      district.Code,
			"city":          district.City,
			"point_count":   len(district.Points),
			"geometry_type": "hull",
			"hull_type":     "convex",
		}

		district.Hull = feature
	}

	return nil
}

func (dm *DistrictManager) SaveDistrictHulls(districts map[string]*District) error {
	// Create features collection
	features := make([]*geojson.Feature, 0, len(districts))
	for _, district := range districts {
		if district.Hull != nil {
			features = append(features, district.Hull)
		}
	}

	// Create feature collection
	fc := geojson.NewFeatureCollection()
	fc.Features = features

	// Add metadata
	metadata := map[string]interface{}{
		"generated":   time.Now().Format(time.RFC3339),
		"description": "District boundaries generated from PDOK postal code coordinates",
		"districts":   len(features),
	}

	// Create the final GeoJSON structure
	output := map[string]interface{}{
		"type":     "FeatureCollection",
		"features": features,
		"metadata": metadata,
	}

	// Ensure the public directory exists
	publicDir := filepath.Join("..", "client", "public")
	if err := os.MkdirAll(publicDir, 0755); err != nil {
		return fmt.Errorf("failed to create public directory: %v", err)
	}

	// Save to file
	outputPath := filepath.Join(publicDir, "district_hulls.geojson")
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("failed to encode GeoJSON: %v", err)
	}

	dm.logger.Infof("Saved %d district hulls to %s", len(features), outputPath)
	return nil
}

func (dm *DistrictManager) UpdateDistrictHulls() error {
	// Get unique districts
	districts, err := dm.GetUniqueDistricts()
	if err != nil {
		return fmt.Errorf("failed to get unique districts: %v", err)
	}

	// Create GeoJSON structure for Python script
	features := []map[string]interface{}{}

	// Fetch points for each district
	for districtCode, city := range districts {
		points, err := dm.FetchDistrictPoints(districtCode, city)
		if err != nil {
			dm.logger.Warnf("Failed to fetch points for district %s: %v", districtCode, err)
			continue
		}

		if len(points) < 3 {
			dm.logger.Warnf("Not enough points for district %s (minimum 3 required)", districtCode)
			continue
		}

		// Convert points to coordinates array
		coordinates := make([][]float64, len(points))
		for i, p := range points {
			coordinates[i] = []float64{p.Longitude, p.Latitude}
		}

		// Create feature for this district
		feature := map[string]interface{}{
			"type": "Feature",
			"geometry": map[string]interface{}{
				"type":        "MultiPoint",
				"coordinates": coordinates,
			},
			"properties": map[string]interface{}{
				"district":    districtCode,
				"city":        city,
				"point_count": len(points),
			},
		}
		features = append(features, feature)
	}

	// Create complete GeoJSON object
	geojson := map[string]interface{}{
		"type":     "FeatureCollection",
		"features": features,
		"metadata": map[string]interface{}{
			"generated": time.Now().Format(time.RFC3339),
			"source":    "PDOK Locatieserver",
		},
	}

	// Convert to JSON
	input, err := json.Marshal(geojson)
	if err != nil {
		return fmt.Errorf("failed to marshal GeoJSON: %v", err)
	}

	// Get the path to the Python script
	scriptPath := filepath.Join("scripts", "generate_hulls.py")

	// Create command
	cmd := exec.Command("python3", scriptPath)
	cmd.Dir = filepath.Dir(filepath.Dir(scriptPath)) // Set working directory to server root

	// Set up pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Python script: %v", err)
	}

	// Write input to stdin
	if _, err := stdin.Write(input); err != nil {
		return fmt.Errorf("failed to write to stdin: %v", err)
	}
	stdin.Close()

	// Read response
	response, err := io.ReadAll(stdout)
	if err != nil {
		return fmt.Errorf("failed to read script output: %v", err)
	}

	// Wait for the command to finish
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("Python script failed: %v", err)
	}

	// Parse response
	var result struct {
		Status    string `json:"status"`
		HullCount int    `json:"hull_count"`
	}
	if err := json.Unmarshal(response, &result); err != nil {
		return fmt.Errorf("failed to parse script response: %v", err)
	}

	dm.logger.Infof("Successfully generated %d district hulls", result.HullCount)
	return nil
}
