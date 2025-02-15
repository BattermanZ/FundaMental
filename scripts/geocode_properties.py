import sqlite3
import requests
import time
import os
import logging
from typing import Optional, Tuple
import json
from pathlib import Path

# Set up logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class PropertyGeocoder:
    def __init__(self, db_path):
        self.db_path = db_path
        self.base_url = "https://nominatim.openstreetmap.org/search"
        self.headers = {
            'User-Agent': 'FundaMental Property Analyzer/1.0 (https://github.com/yourusername/fundamental)',
            'Accept-Language': 'nl-NL,nl;q=0.9,en-US;q=0.8,en;q=0.7'
        }
        
        # Set up cache directory
        self.cache_dir = Path(os.path.dirname(db_path)) / 'geocode_cache'
        self.cache_dir.mkdir(exist_ok=True)
        self.cache_file = self.cache_dir / 'nominatim_cache.json'
        self.load_cache()

    def load_cache(self):
        """Load the geocoding cache from file."""
        if self.cache_file.exists():
            try:
                with open(self.cache_file, 'r') as f:
                    self.cache = json.load(f)
                logger.info(f"Loaded {len(self.cache)} cached addresses")
            except json.JSONDecodeError:
                self.cache = {}
        else:
            self.cache = {}

    def save_cache(self):
        """Save the geocoding cache to file."""
        with open(self.cache_file, 'w') as f:
            json.dump(self.cache, f)
        logger.info(f"Saved {len(self.cache)} addresses to cache")

    def geocode_address(self, street: str, postal_code: str, city: str) -> Tuple[Optional[float], Optional[float]]:
        """Geocode a single address using Nominatim."""
        # Format the address query
        query = f"{street}, {postal_code}, {city}, Netherlands"
        cache_key = f"{street}|{postal_code}|{city}"

        # Check cache first
        if cache_key in self.cache:
            logger.info(f"Cache hit for address: {query}")
            return self.cache[cache_key]
        
        try:
            # Respect Nominatim's usage policy: max 1 request per second
            time.sleep(1)
            
            params = {
                'q': query,
                'format': 'json',
                'limit': 1,
                'countrycodes': 'nl',
                'addressdetails': 1
            }
            
            response = requests.get(
                self.base_url,
                params=params,
                headers=self.headers
            )
            
            if response.status_code == 200:
                results = response.json()
                if results:
                    lat = float(results[0]['lat'])
                    lon = float(results[0]['lon'])
                    
                    # Cache the result
                    self.cache[cache_key] = (lat, lon)
                    # Save cache periodically (every 10 successful geocodes)
                    if len(self.cache) % 10 == 0:
                        self.save_cache()
                    
                    return lat, lon
            
            logger.warning(f"No results found for address: {query}")
            self.cache[cache_key] = (None, None)  # Cache negative results too
            return None, None
            
        except Exception as e:
            logger.error(f"Error geocoding address {query}: {str(e)}")
            return None, None

    def update_coordinates(self, batch_size: int = 100):
        """Update coordinates for all properties in the database."""
        with sqlite3.connect(self.db_path) as conn:
            cursor = conn.cursor()
            
            # Get all properties without coordinates
            cursor.execute('''
                SELECT id, street, postal_code, city 
                FROM properties 
                WHERE latitude IS NULL OR longitude IS NULL
            ''')
            
            properties = cursor.fetchall()
            total = len(properties)
            logger.info(f"Found {total} properties to geocode")
            
            try:
                # Process in batches to avoid memory issues
                for i in range(0, total, batch_size):
                    batch = properties[i:i + batch_size]
                    logger.info(f"Processing batch {i//batch_size + 1}/{(total + batch_size - 1)//batch_size}")
                    
                    for prop_id, street, postal_code, city in batch:
                        if not all([street, postal_code, city]):
                            logger.warning(f"Skipping property {prop_id} due to missing address components")
                            continue
                        
                        logger.info(f"Geocoding property {i + batch.index((prop_id, street, postal_code, city)) + 1}/{total}: {street}, {postal_code}")
                        lat, lon = self.geocode_address(street, postal_code, city)
                        
                        if lat and lon:
                            cursor.execute('''
                                UPDATE properties 
                                SET latitude = ?, longitude = ? 
                                WHERE id = ?
                            ''', (lat, lon, prop_id))
                            conn.commit()
                            logger.info(f"Updated coordinates for property {prop_id}: {lat}, {lon}")
                        else:
                            logger.warning(f"Could not geocode property {prop_id}")
            finally:
                # Save cache before exiting
                self.save_cache()

def main():
    # Get the project root directory
    project_root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    db_path = os.path.join(project_root, 'database', 'funda.db')
    
    if not os.path.exists(db_path):
        logger.error(f"Database not found at {db_path}")
        return
    
    geocoder = PropertyGeocoder(db_path)
    geocoder.update_coordinates()
    logger.info("Geocoding completed")

if __name__ == "__main__":
    main() 