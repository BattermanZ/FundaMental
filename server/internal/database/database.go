package database

import (
	"database/sql"
	"fundamental/server/internal/models"
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

func (d *Database) GetAllProperties() ([]models.Property, error) {
	rows, err := d.db.Query(`
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
    `)
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

func (d *Database) GetPropertyStats() (models.PropertyStats, error) {
	var stats models.PropertyStats
	err := d.db.QueryRow(`
        SELECT 
            COUNT(*) as total_properties,
            AVG(price) as average_price,
            AVG(CASE 
                WHEN listing_date IS NOT NULL AND selling_date IS NOT NULL 
                THEN julianday(selling_date) - julianday(listing_date) 
                END) as avg_days_to_sell,
            COUNT(CASE WHEN status = 'sold' THEN 1 END) as total_sold,
            AVG(CAST(price AS FLOAT) / NULLIF(living_area, 0)) as price_per_sqm
        FROM properties
        WHERE price > 0
    `).Scan(
		&stats.TotalProperties,
		&stats.AveragePrice,
		&stats.AvgDaysToSell,
		&stats.TotalSold,
		&stats.PricePerSqm,
	)
	return stats, err
}

func (d *Database) GetAreaStats(postalPrefix string) (models.AreaStats, error) {
	var stats models.AreaStats
	err := d.db.QueryRow(`
        SELECT 
            postal_code,
            COUNT(*) as property_count,
            AVG(price) as average_price,
            AVG(CAST(price AS FLOAT) / NULLIF(living_area, 0)) as avg_price_per_sqm
        FROM properties
        WHERE postal_code LIKE ? || '%'
        GROUP BY substr(postal_code, 1, 4)
    `, postalPrefix).Scan(
		&stats.PostalCode,
		&stats.PropertyCount,
		&stats.AveragePrice,
		&stats.AvgPricePerSqm,
	)
	return stats, err
}

func (d *Database) GetRecentSales(limit int) ([]models.Property, error) {
	rows, err := d.db.Query(`
        SELECT id, url, street, neighborhood, property_type, city, postal_code,
               price, year_built, living_area, num_rooms, status, 
               listing_date, selling_date, scraped_at, created_at
        FROM properties
        WHERE status = 'sold'
        ORDER BY selling_date DESC
        LIMIT ?
    `, limit)
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
