import sqlite3
from datetime import datetime

class FundaDB:
    def __init__(self, db_path='funda.db'):
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
                    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
                )
            ''')
            
            # Create index on postal_code for geographic queries
            cursor.execute('CREATE INDEX IF NOT EXISTS idx_postal_code ON properties(postal_code)')
            
            conn.commit()

    def insert_property(self, item):
        """Insert a property into the database."""
        with sqlite3.connect(self.db_path) as conn:
            cursor = conn.cursor()
            
            try:
                cursor.execute('''
                    INSERT OR REPLACE INTO properties 
                    (url, street, city, postal_code, price, year_built, 
                     living_area, num_rooms, status, listing_date, 
                     selling_date, scraped_at)
                    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                ''', (
                    item.get('url'),
                    item.get('street'),
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

    def get_basic_stats(self):
        """Get basic statistics about the properties."""
        with sqlite3.connect(self.db_path) as conn:
            cursor = conn.cursor()
            
            stats = {}
            
            # Total properties
            cursor.execute('SELECT COUNT(*) FROM properties')
            stats['total_properties'] = cursor.fetchone()[0]
            
            # Average price
            cursor.execute('SELECT AVG(price) FROM properties')
            stats['avg_price'] = round(cursor.fetchone()[0] or 0, 2)
            
            # Average time to sell (in days)
            cursor.execute('''
                SELECT AVG(JULIANDAY(selling_date) - JULIANDAY(listing_date))
                FROM properties 
                WHERE listing_date IS NOT NULL 
                AND selling_date IS NOT NULL
            ''')
            stats['avg_days_to_sell'] = round(cursor.fetchone()[0] or 0, 1)
            
            return stats

    def get_properties_by_postal_code(self, postal_code_prefix):
        """Get properties in a specific postal code area."""
        with sqlite3.connect(self.db_path) as conn:
            cursor = conn.cursor()
            
            cursor.execute('''
                SELECT * FROM properties 
                WHERE postal_code LIKE ?
                ORDER BY selling_date DESC
            ''', (f"{postal_code_prefix}%",))
            
            columns = [description[0] for description in cursor.description]
            return [dict(zip(columns, row)) for row in cursor.fetchall()]

    def get_recent_sales(self, limit=10):
        """Get the most recent sales."""
        with sqlite3.connect(self.db_path) as conn:
            cursor = conn.cursor()
            
            cursor.execute('''
                SELECT * FROM properties 
                WHERE selling_date IS NOT NULL
                ORDER BY selling_date DESC 
                LIMIT ?
            ''', (limit,))
            
            columns = [description[0] for description in cursor.description]
            return [dict(zip(columns, row)) for row in cursor.fetchall()] 