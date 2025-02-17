import requests
import json
import os
from pathlib import Path
import logging
from time import sleep
from datetime import datetime

# Set up logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def fetch_amsterdam_postal_codes():
    """
    Fetches postal code district boundaries for Amsterdam from PDOK Locatieserver.
    Creates a GeoJSON file with postal code polygons and metadata.
    """
    logger.info("Fetching postal district data from PDOK Locatieserver...")
    
    # PDOK Locatieserver URL
    base_url = "https://api.pdok.nl/bzk/locatieserver/search/v3_1/free"
    
    # Process each postal district (1011-1019)
    features = []
    districts = {}  # Dictionary to store points by district
    
    for district in range(1011, 1020):
        logger.info(f"Fetching data for district {district}...")
        
        # Search parameters
        params = {
            'q': f'type:postcode AND postcode:{district}* AND woonplaatsnaam:Amsterdam',
            'rows': 100,
            'fl': '*',
            'fq': 'type:postcode'
        }
        
        try:
            response = requests.get(base_url, params=params)
            response.raise_for_status()
            data = response.json()
            
            # Process results
            for doc in data.get('response', {}).get('docs', []):
                if 'centroide_ll' not in doc:
                    continue
                    
                # Extract coordinates from centroide_ll
                coords_str = doc['centroide_ll']
                coords = [float(x.strip()) for x in coords_str.replace('POINT(', '').replace(')', '').split()]
                
                # Add point to district
                district_code = str(district)
                if district_code not in districts:
                    districts[district_code] = []
                districts[district_code].append(coords)
                
            # Add delay to avoid rate limiting
            sleep(0.1)
            
        except requests.exceptions.RequestException as e:
            logger.error(f"Error fetching data for district {district}: {e}")
            continue
    
    # Create a feature for each district with its points
    for district, points in districts.items():
        if not points:
            continue
            
        feature_data = {
            'type': 'Feature',
            'geometry': {
                'type': 'MultiPoint',
                'coordinates': points
            },
            'properties': {
                'district': district,
                'woonplaatsnaam': 'Amsterdam',
                'type': 'postcode-district',
                'bron': 'PDOK Locatieserver',
                'point_count': len(points)
            }
        }
        features.append(feature_data)
        logger.info(f"Processed district: {district} with {len(points)} addresses")
    
    if not features:
        logger.warning("No postal district areas found!")
        return
    
    # Create GeoJSON with metadata
    geojson = {
        'type': 'FeatureCollection',
        'features': features,
        'metadata': {
            'generated': datetime.now().isoformat(),
            'title': 'Amsterdam Postal District Areas',
            'description': 'Point collections for postal districts (PC4) in Amsterdam',
            'source': 'PDOK Locatieserver',
            'total_features': len(features),
            'postal_districts': sorted(list(set(f['properties']['district'] for f in features)))
        }
    }
    
    output_path = Path('client/public/amsterdam_postal_districts.geojson')
    output_path.parent.mkdir(parents=True, exist_ok=True)
    
    with open(output_path, 'w') as f:
        json.dump(geojson, f, indent=2)
    
    logger.info(f"Saved {len(features)} postal district boundaries to {output_path}")
    logger.info(f"Postal districts covered: {', '.join(geojson['metadata']['postal_districts'])}")

if __name__ == '__main__':
    fetch_amsterdam_postal_codes() 