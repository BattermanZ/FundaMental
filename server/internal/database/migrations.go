package database

func (d *Database) RunMigrations() error {
	// Add latitude and longitude columns if they don't exist
	_, err := d.db.Exec(`
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
