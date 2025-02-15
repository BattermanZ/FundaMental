import sqlite3
import os
import logging

# Set up logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def migrate_database():
    # Get the project root directory
    project_root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    db_path = os.path.join(project_root, 'database', 'funda.db')
    
    if not os.path.exists(db_path):
        logger.error(f"Database not found at {db_path}")
        return
    
    logger.info(f"Migrating database at {db_path}")
    
    with sqlite3.connect(db_path) as conn:
        cursor = conn.cursor()
        
        # Check if columns already exist
        cursor.execute("PRAGMA table_info(properties)")
        columns = [column[1] for column in cursor.fetchall()]
        
        # Add latitude column if it doesn't exist
        if 'latitude' not in columns:
            logger.info("Adding latitude column")
            cursor.execute("ALTER TABLE properties ADD COLUMN latitude REAL")
        
        # Add longitude column if it doesn't exist
        if 'longitude' not in columns:
            logger.info("Adding longitude column")
            cursor.execute("ALTER TABLE properties ADD COLUMN longitude REAL")
        
        # Create index on coordinates if it doesn't exist
        cursor.execute('''
            CREATE INDEX IF NOT EXISTS idx_coordinates 
            ON properties(latitude, longitude)
        ''')
        
        conn.commit()
        logger.info("Migration completed successfully")

if __name__ == "__main__":
    migrate_database() 