import sqlite3
from datetime import datetime
import os

class FundaDB:
    def __init__(self, db_path=None):
        if db_path is None:
            # Assuming the script is always run from the project root or server directory
            # Look for the database directory in the current directory or one level up
            if os.path.exists('database'):
                db_dir = 'database'
            elif os.path.exists('../database'):
                db_dir = '../database'
            else:
                db_dir = 'database'  # Default to creating in current directory
                os.makedirs(db_dir, exist_ok=True)
            
            self.db_path = os.path.abspath(os.path.join(db_dir, 'funda.db'))
            print(f"Database path: {self.db_path}")  # Debug print
        else:
            self.db_path = db_path
        
        self.init_db()

    def init_db(self):
        """Initialize the database with the required schema."""
        with sqlite3.connect(self.db_path) as conn:
            cursor = conn.cursor()
            
            # Create properties table
            cursor.execute('''
                CREATE TABLE IF NOT EXISTS properties (
                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                    url TEXT UNIQUE,
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
                    listing_date DATE,
                    selling_date DATE,
                    scraped_at TIMESTAMP,
                    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                    latitude REAL,
                    longitude REAL
                )
            ''')
            
            # Create index on postal_code for geographic queries
            cursor.execute('CREATE INDEX IF NOT EXISTS idx_postal_code ON properties(postal_code)')
            
            # Create index on coordinates for spatial queries
            cursor.execute('CREATE INDEX IF NOT EXISTS idx_coordinates ON properties(latitude, longitude)')
            
            conn.commit()

    def insert_property(self, item):
        """Insert a property into the database."""
        with sqlite3.connect(self.db_path) as conn:
            cursor = conn.cursor()
            
            try:
                cursor.execute('''
                    INSERT OR IGNORE INTO properties 
                    (url, street, neighborhood, property_type, city, postal_code, price, year_built, 
                     living_area, num_rooms, status, listing_date, 
                     selling_date, scraped_at)
                    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                ''', (
                    item.get('url'),
                    item.get('street'),
                    item.get('neighborhood'),
                    item.get('property_type'),
                    item.get('city'),
                    item.get('postal_code'),
                    item.get('price'),
                    item.get('year_built'),
                    item.get('living_area'),
                    item.get('num_rooms'),
                    item.get('status'),
                    item.get('listing_date'),
                    item.get('selling_date'),
                    item.get('scraped_at')
                ))
                conn.commit()
                return True
            except sqlite3.Error as e:
                print(f"Error inserting property: {e}")
                return False

    def get_existing_urls(self):
        """Get all existing URLs from the database."""
        with sqlite3.connect(self.db_path) as conn:
            cursor = conn.cursor()
            cursor.execute('SELECT url FROM properties')
            return {row[0] for row in cursor.fetchall()}  # Return as a set for O(1) lookups 