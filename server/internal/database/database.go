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
            longitude,
            energy_label
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
		var energyLabel sql.NullString

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
			&energyLabel,
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

		// Handle energy_label
		if energyLabel.Valid {
			p.EnergyLabel = energyLabel.String
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
	// Create properties table first
	_, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS properties (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			url TEXT UNIQUE NOT NULL,
			street TEXT,
			neighborhood TEXT,
			property_type TEXT,
			city TEXT,
			postal_code TEXT,
			price INTEGER,
			year_built INTEGER,
			living_area INTEGER,
			num_rooms INTEGER,
			status TEXT,
			listing_date TEXT,
			selling_date TEXT,
			scraped_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			energy_label TEXT,
			republish_count INTEGER DEFAULT 0,
			latitude REAL,
			longitude REAL,
			geocoding_attempted BOOLEAN DEFAULT 0
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create properties table: %v", err)
	}

	// Create property_history table
	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS property_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			property_id INTEGER NOT NULL,
			status TEXT,
			price INTEGER,
			listing_date TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (property_id) REFERENCES properties(id)
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create property_history table: %v", err)
	}

	// Create metropolitan areas table
	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS metropolitan_areas (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			center_lat REAL,
			center_lng REAL,
			zoom_level INTEGER DEFAULT 13,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create metropolitan_areas table: %v", err)
	}

	// Create telegram configuration table
	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS telegram_config (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			bot_token TEXT NOT NULL,
			chat_id TEXT NOT NULL,
			is_enabled BOOLEAN DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create telegram_config table: %v", err)
	}

	// Create metropolitan cities table without the foreign key constraint
	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS metropolitan_cities (
			metropolitan_area_id INTEGER,
			city TEXT NOT NULL,
			lat REAL,
			lng REAL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (metropolitan_area_id, city)
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create metropolitan_cities table: %v", err)
	}

	// Add coordinate columns to metropolitan_areas if they don't exist
	_, err = d.db.Exec(`
		ALTER TABLE metropolitan_areas 
		ADD COLUMN center_lat REAL;
	`)
	if err != nil && err.Error() != "duplicate column name: center_lat" {
		return err
	}

	_, err = d.db.Exec(`
		ALTER TABLE metropolitan_areas 
		ADD COLUMN center_lng REAL;
	`)
	if err != nil && err.Error() != "duplicate column name: center_lng" {
		return err
	}

	_, err = d.db.Exec(`
		ALTER TABLE metropolitan_areas 
		ADD COLUMN zoom_level INTEGER DEFAULT 13;
	`)
	if err != nil && err.Error() != "duplicate column name: zoom_level" {
		return err
	}

	// Add coordinate columns to metropolitan_cities if they don't exist
	_, err = d.db.Exec(`
		ALTER TABLE metropolitan_cities 
		ADD COLUMN lat REAL;
	`)
	if err != nil && err.Error() != "duplicate column name: lat" {
		return err
	}

	_, err = d.db.Exec(`
		ALTER TABLE metropolitan_cities 
		ADD COLUMN lng REAL;
	`)
	if err != nil && err.Error() != "duplicate column name: lng" {
		return err
	}

	// Add republish_count column if it doesn't exist
	_, err = d.db.Exec(`
		ALTER TABLE properties 
		ADD COLUMN republish_count INTEGER DEFAULT 0;
	`)
	if err != nil && err.Error() != "duplicate column name: republish_count" {
		return fmt.Errorf("failed to add republish_count column: %v", err)
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

	// Add energy_label column if it doesn't exist
	_, err = d.db.Exec(`
		ALTER TABLE properties 
		ADD COLUMN energy_label TEXT;
	`)
	if err != nil && err.Error() != "duplicate column name: energy_label" {
		return fmt.Errorf("failed to add energy_label column: %v", err)
	}

	// Create telegram_filters table
	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS telegram_filters (
			min_price INTEGER,
			max_price INTEGER,
			min_living_area INTEGER,
			max_living_area INTEGER,
			min_rooms INTEGER,
			max_rooms INTEGER,
			districts TEXT,
			energy_labels TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create telegram_filters table: %v", err)
	}

	// Ensure we have exactly one row in telegram_filters
	var count int
	err = d.db.QueryRow("SELECT COUNT(*) FROM telegram_filters").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to count telegram_filters: %v", err)
	}

	if count == 0 {
		_, err = d.db.Exec("INSERT INTO telegram_filters DEFAULT VALUES")
		if err != nil {
			return fmt.Errorf("failed to insert default telegram_filters: %v", err)
		}
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

// InsertProperties inserts a batch of properties into the database and returns the newly inserted ones
func (d *Database) InsertProperties(properties []map[string]interface{}) ([]map[string]interface{}, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var newProperties []map[string]interface{}

	for _, prop := range properties {
		// Check if property exists and get its current state
		var existingID int64
		var currentStatus string
		var republishCount int
		err = tx.QueryRow(`
			SELECT id, status, republish_count 
			FROM properties 
			WHERE url = ?
		`, prop["url"]).Scan(&existingID, &currentStatus, &republishCount)

		if err == nil {
			// Property exists, handle update
			if currentStatus == "inactive" && prop["status"] == "active" {
				// Property is being republished
				republishCount++
				prop["status"] = "republished"
				prop["republish_count"] = republishCount
			}

			// Update the property
			_, err = tx.Exec(`
				UPDATE properties 
				SET street = ?, 
					neighborhood = ?,
					property_type = ?,
					city = ?,
					postal_code = ?,
					price = ?,
					year_built = ?,
					living_area = CASE WHEN CAST(? AS INTEGER) > 0 THEN CAST(? AS INTEGER) ELSE NULL END,
					num_rooms = ?,
					status = ?,
					listing_date = ?,
					selling_date = ?,
					scraped_at = ?,
					republish_count = ?,
					energy_label = ?
				WHERE url = ?
			`,
				prop["street"],
				prop["neighborhood"],
				prop["property_type"],
				prop["city"],
				prop["postal_code"],
				prop["price"],
				prop["year_built"],
				prop["living_area"], prop["living_area"], // Pass living_area twice for the CASE statement
				prop["num_rooms"],
				prop["status"],
				prop["listing_date"],
				prop["selling_date"],
				prop["scraped_at"],
				republishCount,
				prop["energy_label"],
				prop["url"],
			)
			if err != nil {
				return nil, fmt.Errorf("failed to update property: %w", err)
			}

			// Record history
			_, err = tx.Exec(`
				INSERT INTO property_history 
				(property_id, status, price, listing_date)
				VALUES (?, ?, ?, ?)
			`,
				existingID,
				prop["status"],
				prop["price"],
				prop["listing_date"],
			)
			if err != nil {
				return nil, fmt.Errorf("failed to insert property history: %w", err)
			}

		} else if err == sql.ErrNoRows {
			// Insert new property
			result, err := tx.Exec(`
				INSERT INTO properties 
				(url, street, neighborhood, property_type, city, postal_code, 
				 price, year_built, living_area, num_rooms, status, 
				 listing_date, selling_date, scraped_at, republish_count, energy_label)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, 
				 CASE WHEN CAST(? AS INTEGER) > 0 THEN CAST(? AS INTEGER) ELSE NULL END,
				 ?, ?, ?, ?, ?, ?, ?)
			`,
				prop["url"],
				prop["street"],
				prop["neighborhood"],
				prop["property_type"],
				prop["city"],
				prop["postal_code"],
				prop["price"],
				prop["year_built"],
				prop["living_area"], prop["living_area"], // Pass living_area twice for the CASE statement
				prop["num_rooms"],
				prop["status"],
				prop["listing_date"],
				prop["selling_date"],
				prop["scraped_at"],
				0, // Initial republish_count
				prop["energy_label"],
			)
			if err != nil {
				return nil, fmt.Errorf("failed to insert property: %w", err)
			}

			// Get the new property ID
			propertyID, err := result.LastInsertId()
			if err != nil {
				return nil, fmt.Errorf("failed to get last insert ID: %w", err)
			}

			// Record initial history
			_, err = tx.Exec(`
				INSERT INTO property_history 
				(property_id, status, price, listing_date)
				VALUES (?, ?, ?, ?)
			`,
				propertyID,
				prop["status"],
				prop["price"],
				prop["listing_date"],
			)
			if err != nil {
				return nil, fmt.Errorf("failed to insert initial property history: %w", err)
			}

			newProperties = append(newProperties, prop)
		} else {
			return nil, fmt.Errorf("failed to check existing property: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return newProperties, nil
}

// GetMetropolitanAreas returns all metropolitan areas with their coordinates
func (d *Database) GetMetropolitanAreas() ([]models.MetropolitanArea, error) {
	rows, err := d.db.Query(`
		SELECT m.id, m.name, m.center_lat, m.center_lng, m.zoom_level,
		       GROUP_CONCAT(mc.city) as cities,
		       GROUP_CONCAT(mc.lat) as city_lats,
		       GROUP_CONCAT(mc.lng) as city_lngs
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
		var citiesStr, latStr, lngStr sql.NullString
		if err := rows.Scan(
			&area.ID,
			&area.Name,
			&area.CenterLat,
			&area.CenterLng,
			&area.ZoomLevel,
			&citiesStr,
			&latStr,
			&lngStr,
		); err != nil {
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

// CalculateMetropolitanCenter calculates and updates the geometric center of a metropolitan area
func (d *Database) CalculateMetropolitanCenter(areaID int64) error {
	rows, err := d.db.Query(`
		SELECT lat, lng
		FROM metropolitan_cities
		WHERE metropolitan_area_id = ? AND lat IS NOT NULL AND lng IS NOT NULL
	`, areaID)
	if err != nil {
		return fmt.Errorf("failed to query city coordinates: %v", err)
	}
	defer rows.Close()

	var sumLat, sumLng float64
	var count int

	for rows.Next() {
		var lat, lng float64
		if err := rows.Scan(&lat, &lng); err != nil {
			return fmt.Errorf("failed to scan coordinates: %v", err)
		}
		sumLat += lat
		sumLng += lng
		count++
	}

	if count == 0 {
		return fmt.Errorf("no valid coordinates found for metropolitan area %d", areaID)
	}

	centerLat := sumLat / float64(count)
	centerLng := sumLng / float64(count)

	_, err = d.db.Exec(`
		UPDATE metropolitan_areas
		SET center_lat = ?, center_lng = ?
		WHERE id = ?
	`, centerLat, centerLng, areaID)
	if err != nil {
		return fmt.Errorf("failed to update metropolitan center: %v", err)
	}

	return nil
}

// UpdateCityCoordinates updates the coordinates for a city in a metropolitan area
func (d *Database) UpdateCityCoordinates(areaID int64, city string, lat, lng float64) error {
	_, err := d.db.Exec(`
		UPDATE metropolitan_cities
		SET lat = ?, lng = ?
		WHERE metropolitan_area_id = ? AND city = ?
	`, lat, lng, areaID, city)
	if err != nil {
		return fmt.Errorf("failed to update city coordinates: %v", err)
	}

	return d.CalculateMetropolitanCenter(areaID)
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
		result, err := tx.Exec(`
			INSERT INTO metropolitan_areas (name, center_lat, center_lng, zoom_level) 
			VALUES (?, ?, ?, ?)
		`, area.Name, area.CenterLat, area.CenterLng, area.ZoomLevel)
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
		_, err = tx.Exec(`
			UPDATE metropolitan_areas 
			SET center_lat = ?, center_lng = ?, zoom_level = ?
			WHERE id = ?
		`, area.CenterLat, area.CenterLng, area.ZoomLevel, id)
		if err != nil {
			return fmt.Errorf("failed to update metropolitan area: %v", err)
		}
	}

	// Delete existing cities for this metropolitan area
	_, err = tx.Exec("DELETE FROM metropolitan_cities WHERE metropolitan_area_id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete existing cities: %v", err)
	}

	// Insert new cities
	for _, city := range area.Cities {
		_, err = tx.Exec(`
			INSERT INTO metropolitan_cities (metropolitan_area_id, city, lat, lng)
			VALUES (?, ?, ?, ?)
		`, id, city, nil, nil) // Coordinates will be updated by geocoding service
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

// GetTelegramConfig returns the current Telegram configuration
func (d *Database) GetTelegramConfig() (*models.TelegramConfig, error) {
	var config models.TelegramConfig
	err := d.db.QueryRow(`
		SELECT id, bot_token, chat_id, is_enabled, created_at, updated_at
		FROM telegram_config
		ORDER BY id DESC
		LIMIT 1
	`).Scan(
		&config.ID,
		&config.BotToken,
		&config.ChatID,
		&config.IsEnabled,
		&config.CreatedAt,
		&config.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get telegram config: %v", err)
	}
	return &config, nil
}

// UpdateTelegramConfig updates or creates the Telegram configuration
func (d *Database) UpdateTelegramConfig(config *models.TelegramConfigRequest) error {
	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO telegram_config
		(bot_token, chat_id, is_enabled, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
	`,
		config.BotToken,
		config.ChatID,
		config.IsEnabled,
	)
	if err != nil {
		return fmt.Errorf("failed to update telegram config: %v", err)
	}
	return nil
}

// GetDistrictMedianPricePerSqm returns the median price per square meter for a district (4-digit postal code)
func (d *Database) GetDistrictMedianPricePerSqm(district string) (float64, error) {
	query := `
		WITH prices_per_sqm AS (
			SELECT 
				CAST(price AS FLOAT) / CAST(living_area AS FLOAT) as price_per_sqm
			FROM properties 
			WHERE substr(postal_code, 1, 4) = ?
				AND price > 0 
				AND living_area > 0
				AND selling_date IS NOT NULL
				AND selling_date >= date('now', '-1 year')
		)
		SELECT 
			AVG(price_per_sqm) as median_price
		FROM (
			SELECT price_per_sqm
			FROM prices_per_sqm
			ORDER BY price_per_sqm
			LIMIT 2 - (SELECT COUNT(*) FROM prices_per_sqm) % 2
			OFFSET (SELECT (COUNT(*) - 1) / 2 FROM prices_per_sqm)
		);
	`

	var medianPrice *float64
	err := d.db.QueryRow(query, district).Scan(&medianPrice)
	if err == sql.ErrNoRows || medianPrice == nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get median price per sqm: %v", err)
	}

	return *medianPrice, nil
}

// MarkInactiveProperties marks properties as inactive if their URLs are not in the activeURLs list
func (d *Database) MarkInactiveProperties(city string, activeURLs []string) error {
	// Convert activeURLs slice to a map for O(1) lookup
	activeURLMap := make(map[string]bool)
	for _, url := range activeURLs {
		activeURLMap[url] = true
	}

	// Start a transaction
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Get all active properties for the city
	rows, err := tx.Query(`
		SELECT id, url FROM properties 
		WHERE city = ? AND status = 'active'
	`, city)
	if err != nil {
		return fmt.Errorf("failed to query active properties: %v", err)
	}
	defer rows.Close()

	// Collect properties to mark as inactive
	var inactiveIDs []int64
	for rows.Next() {
		var id int64
		var url string
		if err := rows.Scan(&id, &url); err != nil {
			return fmt.Errorf("failed to scan row: %v", err)
		}

		// If URL is not in activeURLs, mark for update
		if !activeURLMap[url] {
			inactiveIDs = append(inactiveIDs, id)
		}
	}
	if err = rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %v", err)
	}

	// Update properties in batches
	if len(inactiveIDs) > 0 {
		// Convert IDs to string for the IN clause
		idStr := make([]string, len(inactiveIDs))
		idArgs := make([]interface{}, len(inactiveIDs))
		for i, id := range inactiveIDs {
			idStr[i] = "?"
			idArgs[i] = id
		}

		query := fmt.Sprintf(`
			UPDATE properties 
			SET status = 'inactive', 
				updated_at = CURRENT_TIMESTAMP 
			WHERE id IN (%s)
		`, strings.Join(idStr, ","))

		_, err = tx.Exec(query, idArgs...)
		if err != nil {
			return fmt.Errorf("failed to update inactive properties: %v", err)
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}

// GetDistrictPriceAnalysis returns median prices and counts for both active and sold properties
func (d *Database) GetDistrictPriceAnalysis(district string) (activeMedian float64, activeCount int, soldMedian float64, soldCount int, err error) {
	// Get active listings median and count
	err = d.db.QueryRow(`
		WITH price_per_sqm AS (
			SELECT 
				price / living_area as price_sqm,
				COUNT(*) OVER () as total_count
			FROM properties
			WHERE substr(postal_code, 1, 4) = ?
			AND status = 'active'
			AND price > 0 AND living_area > 0
			-- Additional data quality checks
			AND living_area BETWEEN 15 AND 1000  -- Reasonable size range
			AND price BETWEEN 50000 AND 10000000  -- Reasonable price range
		),
		ranked AS (
			SELECT 
				price_sqm,
				ROW_NUMBER() OVER (ORDER BY price_sqm) as row_num,
				total_count
			FROM price_per_sqm
		)
		SELECT 
			COALESCE(
				CASE 
					WHEN total_count = 0 THEN 0
					WHEN total_count % 2 = 0 THEN
						-- Even number of rows: average of two middle values
						(SELECT AVG(price_sqm) 
						 FROM ranked 
						 WHERE row_num IN ((total_count/2), (total_count/2) + 1))
					ELSE
						-- Odd number of rows: middle value
						(SELECT price_sqm 
						 FROM ranked 
						 WHERE row_num = (total_count + 1)/2)
				END, 0
			) as median,
			MAX(total_count) as count
		FROM ranked
	`, district).Scan(&activeMedian, &activeCount)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to get active listings analysis: %v", err)
	}

	// Get sold properties median and count (last 12 months)
	err = d.db.QueryRow(`
		WITH price_per_sqm AS (
			SELECT 
				price / living_area as price_sqm,
				COUNT(*) OVER () as total_count
			FROM properties
			WHERE substr(postal_code, 1, 4) = ?
			AND status = 'sold'
			AND price > 0 AND living_area > 0
			-- Additional data quality checks
			AND living_area BETWEEN 15 AND 1000  -- Reasonable size range
			AND price BETWEEN 50000 AND 10000000  -- Reasonable price range
			AND selling_date >= date('now', '-12 months')
		),
		ranked AS (
			SELECT 
				price_sqm,
				ROW_NUMBER() OVER (ORDER BY price_sqm) as row_num,
				total_count
			FROM price_per_sqm
		)
		SELECT 
			COALESCE(
				CASE 
					WHEN total_count = 0 THEN 0
					WHEN total_count % 2 = 0 THEN
						-- Even number of rows: average of two middle values
						(SELECT AVG(price_sqm) 
						 FROM ranked 
						 WHERE row_num IN ((total_count/2), (total_count/2) + 1))
					ELSE
						-- Odd number of rows: middle value
						(SELECT price_sqm 
						 FROM ranked 
						 WHERE row_num = (total_count + 1)/2)
				END, 0
			) as median,
			MAX(total_count) as count
		FROM ranked
	`, district).Scan(&soldMedian, &soldCount)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to get sold properties analysis: %v", err)
	}

	return activeMedian, activeCount, soldMedian, soldCount, nil
}

// GetPreviousPrice returns the previous price for a property
func (d *Database) GetPreviousPrice(propertyID int64) (int, error) {
	var previousPrice int
	err := d.db.QueryRow(`
		SELECT price
		FROM property_history
		WHERE property_id = ?
		ORDER BY listing_date DESC
		LIMIT 1 OFFSET 1
	`, propertyID).Scan(&previousPrice)

	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get previous price: %v", err)
	}

	return previousPrice, nil
}

// GetTelegramFilters retrieves the current telegram notification filters
func (d *Database) GetTelegramFilters() (*models.TelegramFilters, error) {
	filters := &models.TelegramFilters{}
	var districts, energyLabels sql.NullString

	err := d.db.QueryRow(`
		SELECT 
			min_price, max_price,
			min_living_area, max_living_area,
			min_rooms, max_rooms,
			districts, energy_labels
		FROM telegram_filters LIMIT 1
	`).Scan(
		&filters.MinPrice, &filters.MaxPrice,
		&filters.MinLivingArea, &filters.MaxLivingArea,
		&filters.MinRooms, &filters.MaxRooms,
		&districts, &energyLabels,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get telegram filters: %v", err)
	}

	// Convert string arrays from database
	if districts.Valid && districts.String != "" {
		filters.Districts = strings.Split(districts.String, ",")
	}
	if energyLabels.Valid && energyLabels.String != "" {
		filters.EnergyLabels = strings.Split(energyLabels.String, ",")
	}

	return filters, nil
}

// UpdateTelegramFilters updates the telegram notification filters
func (d *Database) UpdateTelegramFilters(filters *models.TelegramFilters) error {
	var districts, energyLabels sql.NullString

	// Convert string arrays to database format
	if len(filters.Districts) > 0 {
		districts = sql.NullString{String: strings.Join(filters.Districts, ","), Valid: true}
	}
	if len(filters.EnergyLabels) > 0 {
		energyLabels = sql.NullString{String: strings.Join(filters.EnergyLabels, ","), Valid: true}
	}

	_, err := d.db.Exec(`
		UPDATE telegram_filters SET
			min_price = $1,
			max_price = $2,
			min_living_area = $3,
			max_living_area = $4,
			min_rooms = $5,
			max_rooms = $6,
			districts = $7,
			energy_labels = $8
	`, filters.MinPrice, filters.MaxPrice,
		filters.MinLivingArea, filters.MaxLivingArea,
		filters.MinRooms, filters.MaxRooms,
		districts, energyLabels)

	if err != nil {
		return fmt.Errorf("failed to update telegram filters: %v", err)
	}

	return nil
}
