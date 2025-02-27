import sqlite3
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

    def get_property_status(self, url):
        """Get the current status of a property."""
        with sqlite3.connect(self.db_path) as conn:
            cursor = conn.cursor()
            cursor.execute('SELECT status FROM properties WHERE url = ?', (url,))
            result = cursor.fetchone()
            return result[0] if result else None

    def get_existing_urls(self):
        """
        DEPRECATED: Use get_sold_urls() or get_all_active_urls() instead.
        This method is kept for backward compatibility.
        """
        return self.get_sold_urls()

    def get_sold_urls(self):
        """Get URLs of properties that are already marked as sold."""
        with sqlite3.connect(self.db_path) as conn:
            cursor = conn.cursor()
            cursor.execute('SELECT url FROM properties WHERE status = "sold"')
            urls = {row[0] for row in cursor.fetchall()}
            print(f"Found {len(urls)} sold URLs in database")  # Keep this useful log
            return urls

    def get_all_active_urls(self):
        """Get URLs of all properties that are either active, inactive, or republished (not sold)."""
        with sqlite3.connect(self.db_path) as conn:
            cursor = conn.cursor()
            cursor.execute('SELECT url FROM properties WHERE status IN ("active", "inactive", "republished")')
            return {row[0] for row in cursor.fetchall()} 