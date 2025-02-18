package database

import (
	"database/sql"
	"fmt"
	"fundamental/server/internal/geocoding"
	"fundamental/server/internal/models"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	db *sql.DB
}

func NewDatabase(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Enable foreign keys
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		return nil, err
	}

	return &Database{db: db}, nil
}

func (d *Database) GetAllProperties(startDate, endDate string, city string) ([]models.Property, error) {
	query := `
        SELECT 
            id, 
            url, 
            street, 
            neighborhood, 
            property_type, 
            city, 
            postal_code,
            price, 
            year_built, 
            living_area, 
            num_rooms, 
            status,
            COALESCE(listing_date, '') as listing_date, 
            COALESCE(selling_date, '') as selling_date,
            COALESCE(scraped_at, CURRENT_TIMESTAMP) as scraped_at,
            COALESCE(created_at, CURRENT_TIMESTAMP) as created_at,
            latitude,
            longitude
        FROM properties
        WHERE (
            -- For active properties, check effective_date (listing_date or scraped_at)
            (status = 'active' AND (
                ? = '' OR COALESCE(listing_date, scraped_at) >= ?
            ) AND (
                ? = '' OR COALESCE(listing_date, scraped_at) <= ?
            ))
            OR
            -- For sold properties, check selling_date only if it exists
            (status = 'sold' AND selling_date IS NOT NULL AND (
                ? = '' OR selling_date >= ?
            ) AND (
                ? = '' OR selling_date <= ?
            ))
        )
        AND (? = '' OR LOWER(city) = LOWER(?))
    `
	var args []interface{}
	args = append(args,
		startDate, startDate, // For active properties listing_date >= ?
		endDate, endDate, // For active properties listing_date <= ?
		startDate, startDate, // For sold properties selling_date >= ?
		endDate, endDate, // For sold properties selling_date <= ?
		city, city, // For city filter
	)

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var properties []models.Property
	for rows.Next() {
		var p models.Property
		var street, neighborhood, propertyType, city, postalCode, status sql.NullString
		var listingDate, sellingDate, scrapedAt, createdAt sql.NullString
		var yearBuilt, livingArea, numRooms sql.NullInt64
		var price sql.NullInt64
		var latitude, longitude sql.NullFloat64

		err := rows.Scan(
			&p.ID,
			&p.URL,
			&street,
			&neighborhood,
			&propertyType,
			&city,
			&postalCode,
			&price,
			&yearBuilt,
			&livingArea,
			&numRooms,
			&status,
			&listingDate,
			&sellingDate,
			&scrapedAt,
			&createdAt,
			&latitude,
			&longitude,
		)
		if err != nil {
			return nil, err
		}

		// Handle nullable string fields
		if street.Valid {
			p.Street = street.String
		}
		if neighborhood.Valid {
			p.Neighborhood = neighborhood.String
		}
		if propertyType.Valid {
			p.PropertyType = propertyType.String
		}
		if city.Valid {
			p.City = city.String
		}
		if postalCode.Valid {
			p.PostalCode = postalCode.String
		}
		if status.Valid {
			p.Status = status.String
		}

		// Handle nullable numeric fields
		if price.Valid {
			p.Price = int(price.Int64)
		}
		if yearBuilt.Valid {
			yb := int(yearBuilt.Int64)
			p.YearBuilt = &yb
		}
		if livingArea.Valid {
			la := int(livingArea.Int64)
			p.LivingArea = &la
		}
		if numRooms.Valid {
			nr := int(numRooms.Int64)
			p.NumRooms = &nr
		}

		// Handle nullable coordinates
		if latitude.Valid {
			lat := latitude.Float64
			p.Latitude = &lat
		}
		if longitude.Valid {
			lon := longitude.Float64
			p.Longitude = &lon
		}

		// Parse dates if they're valid
		if listingDate.Valid && listingDate.String != "" {
			if t, err := time.Parse("2006-01-02", listingDate.String); err == nil {
				p.ListingDate = t
			}
		}
		if sellingDate.Valid && sellingDate.String != "" {
			if t, err := time.Parse("2006-01-02", sellingDate.String); err == nil {
				p.SellingDate = t
			}
		}
		if scrapedAt.Valid && scrapedAt.String != "" {
			if t, err := time.Parse(time.RFC3339, scrapedAt.String); err == nil {
				p.ScrapedAt = t
			}
		}
		if createdAt.Valid && createdAt.String != "" {
			if t, err := time.Parse(time.RFC3339, createdAt.String); err == nil {
				p.CreatedAt = t
			}
		}

		properties = append(properties, p)
	}
	return properties, nil
}

func (d *Database) GetPropertyStats(startDate, endDate string, city string) (models.PropertyStats, error) {
	query := `
        WITH price_data AS (
            SELECT 
                price,
                living_area,
                status,
                COALESCE(listing_date, scraped_at) as effective_date,
                selling_date,
                CASE 
                    WHEN listing_date IS NOT NULL AND selling_date IS NOT NULL 
                    THEN julianday(selling_date) - julianday(listing_date) 
                END as days_to_sell
            FROM properties
            WHERE price IS NOT NULL
            AND (? = '' OR LOWER(city) = LOWER(?))
            AND (
                -- For active properties, check effective_date (listing_date or scraped_at)
                (status = 'active' AND (
                    ? = '' OR COALESCE(listing_date, scraped_at) >= ?
                ) AND (
                    ? = '' OR COALESCE(listing_date, scraped_at) <= ?
                ))
                OR
                -- For sold properties, check selling_date only if it exists
                (status = 'sold' AND selling_date IS NOT NULL AND (
                    ? = '' OR selling_date >= ?
                ) AND (
                    ? = '' OR selling_date <= ?
                ))
            )
        ),
        active_stats AS (
            SELECT 
                COUNT(*) as active_count,
                COALESCE(AVG(price), 0) as active_avg_price,
                COALESCE(AVG(CAST(price AS FLOAT) / NULLIF(living_area, 0)), 0) as active_price_per_sqm
            FROM price_data
            WHERE status = 'active'
        ),
        sold_stats AS (
            SELECT 
                COUNT(*) as sold_count,
                COALESCE(AVG(price), 0) as sold_avg_price,
                COALESCE(AVG(days_to_sell), 0) as avg_days_to_sell,
                COALESCE(AVG(CAST(price AS FLOAT) / NULLIF(living_area, 0)), 0) as sold_price_per_sqm
            FROM price_data
            WHERE status = 'sold'
        )
        SELECT 
            COALESCE(active_count + sold_count, 0) as total_properties,
            CASE 
                WHEN (active_count + sold_count) > 0 
                THEN ROUND(COALESCE(((active_avg_price * active_count) + (sold_avg_price * sold_count)) / NULLIF((active_count + sold_count), 0), 0))
                ELSE 0 
            END as average_price,
            CASE 
                WHEN (active_count + sold_count) > 0 
                THEN ROUND(COALESCE(((active_price_per_sqm * active_count) + (sold_price_per_sqm * sold_count)) / NULLIF((active_count + sold_count), 0), 0))
                ELSE 0 
            END as price_per_sqm,
            COALESCE(avg_days_to_sell, 0) as avg_days_to_sell,
            COALESCE(sold_count, 0) as total_sold,
            COALESCE(active_count, 0) as total_active
        FROM active_stats, sold_stats
    `
	var args []interface{}
	args = append(args,
		city, city, // For city filter
		startDate, startDate, // For active properties listing_date >= ?
		endDate, endDate, // For active properties listing_date <= ?
		startDate, startDate, // For sold properties selling_date >= ?
		endDate, endDate, // For sold properties selling_date <= ?
	)

	var stats models.PropertyStats
	err := d.db.QueryRow(query, args...).Scan(
		&stats.TotalProperties,
		&stats.AveragePrice,
		&stats.PricePerSqm,
		&stats.AvgDaysToSell,
		&stats.TotalSold,
		&stats.TotalActive,
	)
	return stats, err
}

func (d *Database) GetAreaStats(postalPrefix string, startDate, endDate string, city string) (models.AreaStats, error) {
	query := `
        SELECT 
            postal_code,
            COUNT(*) as property_count,
            AVG(price) as average_price,
            AVG(CAST(price AS FLOAT) / NULLIF(living_area, 0)) as avg_price_per_sqm
        FROM properties
        WHERE postal_code LIKE ? || '%'
        AND (? = '' OR LOWER(city) = LOWER(?))
        AND (
            -- For active properties, check effective_date (listing_date or scraped_at)
            (status = 'active' AND (
                ? = '' OR COALESCE(listing_date, scraped_at) >= ?
            ) AND (
                ? = '' OR COALESCE(listing_date, scraped_at) <= ?
            ))
            OR
            -- For sold properties, check selling_date only if it exists
            (status = 'sold' AND selling_date IS NOT NULL AND (
                ? = '' OR selling_date >= ?
            ) AND (
                ? = '' OR selling_date <= ?
            ))
        )
        GROUP BY substr(postal_code, 1, 4)
    `
	var args []interface{}
	args = append(args,
		postalPrefix,
		city, city, // For city filter
		startDate, startDate, // For active properties listing_date >= ?
		endDate, endDate, // For active properties listing_date <= ?
		startDate, startDate, // For sold properties selling_date >= ?
		endDate, endDate, // For sold properties selling_date <= ?
	)

	var stats models.AreaStats
	err := d.db.QueryRow(query, args...).Scan(
		&stats.PostalCode,
		&stats.PropertyCount,
		&stats.AveragePrice,
		&stats.AvgPricePerSqm,
	)
	return stats, err
}

func (d *Database) GetRecentSales(limit int, startDate, endDate string, city string) ([]models.Property, error) {
	query := `
        SELECT id, url, street, neighborhood, property_type, city, postal_code,
               price, year_built, living_area, num_rooms, status, 
               listing_date, selling_date, scraped_at, created_at
        FROM properties
        WHERE status = 'sold'
        AND (? = '' OR LOWER(city) = LOWER(?))
    `
	var args []interface{}
	args = append(args, city, city)

	if startDate != "" {
		query += " AND selling_date >= ?"
		args = append(args, startDate)
	}
	if endDate != "" {
		query += " AND selling_date <= ?"
		args = append(args, endDate)
	}

	query += " ORDER BY selling_date DESC LIMIT ?"
	args = append(args, limit)

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var properties []models.Property
	for rows.Next() {
		var p models.Property
		err := rows.Scan(
			&p.ID, &p.URL, &p.Street, &p.Neighborhood, &p.PropertyType,
			&p.City, &p.PostalCode, &p.Price, &p.YearBuilt, &p.LivingArea,
			&p.NumRooms, &p.Status, &p.ListingDate, &p.SellingDate,
			&p.ScrapedAt, &p.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		properties = append(properties, p)
	}
	return properties, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) RunMigrations() error {
	// Create metropolitan areas table
	_, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS metropolitan_areas (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create metropolitan_areas table: %v", err)
	}

	// Create metropolitan cities table without the foreign key constraint
	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS metropolitan_cities (
			metropolitan_area_id INTEGER,
			city TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (metropolitan_area_id, city)
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create metropolitan_cities table: %v", err)
	}

	// Add latitude and longitude columns if they don't exist
	_, err = d.db.Exec(`
		ALTER TABLE properties 
		ADD COLUMN latitude REAL;
	`)
	if err != nil && err.Error() != "duplicate column name: latitude" {
		return err
	}

	_, err = d.db.Exec(`
		ALTER TABLE properties 
		ADD COLUMN longitude REAL;
	`)
	if err != nil && err.Error() != "duplicate column name: longitude" {
		return err
	}

	// Add geocoding_attempted column
	_, err = d.db.Exec(`
		ALTER TABLE properties 
		ADD COLUMN geocoding_attempted BOOLEAN DEFAULT 0;
	`)
	if err != nil && err.Error() != "duplicate column name: geocoding_attempted" {
		return err
	}

	// Mark properties that already have coordinates as attempted
	_, err = d.db.Exec(`
		UPDATE properties 
		SET geocoding_attempted = 1 
		WHERE latitude IS NOT NULL 
		AND longitude IS NOT NULL;
	`)
	if err != nil {
		return fmt.Errorf("failed to mark existing coordinates as attempted: %v", err)
	}

	// Create spatial index on coordinates
	_, err = d.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_properties_coordinates 
		ON properties(latitude, longitude);
	`)
	if err != nil {
		return err
	}

	return nil
}

func (d *Database) UpdateMissingCoordinates(geocoder *geocoding.Geocoder) error {
	// Get total count of properties needing geocoding
	var totalCount int
	err := d.db.QueryRow(`
		SELECT COUNT(*) 
		FROM properties 
		WHERE (latitude IS NULL OR longitude IS NULL)
		AND geocoding_attempted = 0
		AND street IS NOT NULL 
		AND postal_code IS NOT NULL 
		AND city IS NOT NULL
	`).Scan(&totalCount)
	if err != nil {
		return fmt.Errorf("failed to count properties: %v", err)
	}

	if totalCount == 0 {
		fmt.Println("No properties need geocoding")
		return nil
	}

	fmt.Printf("Found %d properties that need geocoding\n", totalCount)

	var processed, failed int
	batchSize := 10

	// Process properties in batches
	for processed+failed < totalCount {
		// Start a new transaction for each batch
		tx, err := d.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %v", err)
		}

		rows, err := tx.Query(`
			SELECT id, street, postal_code, city 
			FROM properties 
			WHERE (latitude IS NULL OR longitude IS NULL)
			AND geocoding_attempted = 0
			AND street IS NOT NULL 
			AND postal_code IS NOT NULL 
			AND city IS NOT NULL
			LIMIT ?
		`, batchSize)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to query properties: %v", err)
		}

		stmt, err := tx.Prepare(`
			UPDATE properties 
			SET latitude = ?, longitude = ?, geocoding_attempted = 1
			WHERE id = ?
		`)
		if err != nil {
			rows.Close()
			tx.Rollback()
			return fmt.Errorf("failed to prepare statement: %v", err)
		}

		failedStmt, err := tx.Prepare(`
			UPDATE properties 
			SET geocoding_attempted = 1
			WHERE id = ?
		`)
		if err != nil {
			rows.Close()
			stmt.Close()
			tx.Rollback()
			return fmt.Errorf("failed to prepare failed statement: %v", err)
		}

		var batchProcessed int
		for rows.Next() {
			var id int64
			var street, postalCode, city string
			if err := rows.Scan(&id, &street, &postalCode, &city); err != nil {
				rows.Close()
				stmt.Close()
				failedStmt.Close()
				tx.Rollback()
				return fmt.Errorf("failed to scan row: %v", err)
			}

			lat, lon, err := geocoder.GeocodeAddress(street, postalCode, city)
			if err != nil {
				fmt.Printf("Failed to geocode %s, %s, %s: %v\n", street, postalCode, city, err)
				// Mark as attempted even if geocoding failed
				_, err = failedStmt.Exec(id)
				if err != nil {
					rows.Close()
					stmt.Close()
					failedStmt.Close()
					tx.Rollback()
					return fmt.Errorf("failed to mark geocoding attempt: %v", err)
				}
				failed++
				batchProcessed++
				continue
			}

			_, err = stmt.Exec(lat, lon, id)
			if err != nil {
				rows.Close()
				stmt.Close()
				failedStmt.Close()
				tx.Rollback()
				return fmt.Errorf("failed to update coordinates: %v", err)
			}

			processed++
			batchProcessed++

			// Print progress
			fmt.Printf("Progress: %d/%d properties processed (%.1f%%), %d failed\n",
				processed+failed, totalCount, float64(processed+failed)/float64(totalCount)*100, failed)
		}

		rows.Close()
		stmt.Close()
		failedStmt.Close()

		// Commit the batch
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %v", err)
		}

		// If we didn't process any items in this batch, something might be wrong
		if batchProcessed == 0 {
			return fmt.Errorf("no properties processed in batch, possible data inconsistency. Total processed: %d/%d",
				processed+failed, totalCount)
		}
	}

	// Log final stats
	fmt.Printf("Geocoding completed: %d/%d properties processed (%.1f%%), %d failed\n",
		processed+failed, totalCount, float64(processed+failed)/float64(totalCount)*100, failed)

	return nil
}

func (d *Database) GetDB() *sql.DB {
	return d.db
}

// InsertProperties inserts a batch of properties into the database
func (d *Database) InsertProperties(properties []map[string]interface{}) error {
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO properties 
		(url, street, neighborhood, property_type, city, postal_code, price, year_built, 
		 living_area, num_rooms, status, listing_date, selling_date, scraped_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, prop := range properties {
		_, err = stmt.Exec(
			prop["url"],
			prop["street"],
			prop["neighborhood"],
			prop["property_type"],
			prop["city"],
			prop["postal_code"],
			prop["price"],
			prop["year_built"],
			prop["living_area"],
			prop["num_rooms"],
			prop["status"],
			prop["listing_date"],
			prop["selling_date"],
			prop["scraped_at"],
		)
		if err != nil {
			return fmt.Errorf("failed to insert property: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetMetropolitanAreas returns all metropolitan areas
func (d *Database) GetMetropolitanAreas() ([]models.MetropolitanArea, error) {
	rows, err := d.db.Query(`
		SELECT m.id, m.name, GROUP_CONCAT(mc.city, ',') as cities
		FROM metropolitan_areas m
		LEFT JOIN metropolitan_cities mc ON m.id = mc.metropolitan_area_id
		GROUP BY m.id, m.name
		ORDER BY m.id
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query metropolitan areas: %v", err)
	}
	defer rows.Close()

	var areas []models.MetropolitanArea
	for rows.Next() {
		var area models.MetropolitanArea
		var citiesStr sql.NullString
		if err := rows.Scan(&area.ID, &area.Name, &citiesStr); err != nil {
			return nil, fmt.Errorf("failed to scan metropolitan area: %v", err)
		}
		if citiesStr.Valid && citiesStr.String != "" {
			area.Cities = strings.Split(citiesStr.String, ",")
		} else {
			area.Cities = []string{}
		}
		areas = append(areas, area)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating metropolitan areas: %v", err)
	}

	return areas, nil
}

// GetMetropolitanAreaByName returns a specific metropolitan area by name
func (d *Database) GetMetropolitanAreaByName(name string) (*models.MetropolitanArea, error) {
	var area models.MetropolitanArea
	var citiesStr sql.NullString

	err := d.db.QueryRow(`
		SELECT m.id, m.name, GROUP_CONCAT(mc.city) as cities
		FROM metropolitan_areas m
		LEFT JOIN metropolitan_cities mc ON m.id = mc.metropolitan_area_id
		WHERE m.name = ?
		GROUP BY m.id, m.name
	`, name).Scan(&area.ID, &area.Name, &citiesStr)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query metropolitan area: %v", err)
	}

	if citiesStr.Valid && citiesStr.String != "" {
		area.Cities = strings.Split(citiesStr.String, ",")
	} else {
		area.Cities = []string{}
	}

	return &area, nil
}

// UpdateMetropolitanArea updates or creates a metropolitan area
func (d *Database) UpdateMetropolitanArea(area models.MetropolitanArea) error {
	// Start a transaction
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Check if the area exists by name
	var existingID int64
	err = tx.QueryRow("SELECT id FROM metropolitan_areas WHERE name = ?", area.Name).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to check existing metropolitan area: %v", err)
	}

	// Insert or update the metropolitan area
	var id int64
	if err == sql.ErrNoRows {
		// Insert new area
		result, err := tx.Exec("INSERT INTO metropolitan_areas (name) VALUES (?)", area.Name)
		if err != nil {
			return fmt.Errorf("failed to insert metropolitan area: %v", err)
		}
		id, err = result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get metropolitan area ID: %v", err)
		}
	} else {
		// Update existing area
		id = existingID
	}

	// Delete existing cities for this metropolitan area
	_, err = tx.Exec("DELETE FROM metropolitan_cities WHERE metropolitan_area_id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete existing cities: %v", err)
	}

	// Insert new cities
	for _, city := range area.Cities {
		_, err = tx.Exec(`
			INSERT INTO metropolitan_cities (metropolitan_area_id, city)
			VALUES (?, ?)
		`, id, city)
		if err != nil {
			return fmt.Errorf("failed to insert city: %v", err)
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}

// DeleteMetropolitanArea deletes a metropolitan area and its cities
func (d *Database) DeleteMetropolitanArea(name string) error {
	result, err := d.db.Exec("DELETE FROM metropolitan_areas WHERE name = ?", name)
	if err != nil {
		return fmt.Errorf("failed to delete metropolitan area: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("metropolitan area not found: %s", name)
	}

	return nil
}

// GetCitiesInMetropolitanArea returns all cities in a metropolitan area
func (d *Database) GetCitiesInMetropolitanArea(name string) ([]string, error) {
	rows, err := d.db.Query(`
		SELECT mc.city
		FROM metropolitan_cities mc
		JOIN metropolitan_areas ma ON mc.metropolitan_area_id = ma.id
		WHERE ma.name = ?
	`, name)
	if err != nil {
		return nil, fmt.Errorf("failed to query cities: %v", err)
	}
	defer rows.Close()

	var cities []string
	for rows.Next() {
		var city string
		if err := rows.Scan(&city); err != nil {
			return nil, fmt.Errorf("failed to scan city: %v", err)
		}
		cities = append(cities, city)
	}

	return cities, nil
}

func (d *Database) cityExists(city string) (bool, error) {
	var exists bool
	err := d.db.QueryRow("SELECT EXISTS(SELECT 1 FROM properties WHERE LOWER(city) = LOWER(?) LIMIT 1)", city).Scan(&exists)
	return exists, err
}
