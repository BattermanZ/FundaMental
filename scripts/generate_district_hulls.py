import json
from pathlib import Path
import logging
from geometry.hull_generator import generate_district_hull

# Set up logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def process_districts():
    """
    Process the point data for each postal district and generate hull polygons.
    Reads from amsterdam_postal_districts.geojson and creates amsterdam_district_hulls.geojson.
    """
    logger.info("Reading postal district point data...")
    
    input_path = Path('client/public/amsterdam_postal_districts.geojson')
    output_path = Path('client/public/amsterdam_district_hulls.geojson')
    
    # Read the input GeoJSON
    with open(input_path, 'r') as f:
        point_data = json.load(f)
    
    # Process each district
    hull_features = []
    for feature in point_data['features']:
        district = feature['properties']['district']
        points = feature['geometry']['coordinates']
        
        logger.info(f"Generating hull for district {district}...")
        
        # Generate hull for the district
        hull_geometry = generate_district_hull(points, buffer_distance=0.001)
        
        if hull_geometry:
            # Create feature with hull geometry
            hull_feature = {
                'type': 'Feature',
                'geometry': hull_geometry,
                'properties': {
                    **feature['properties'],  # Copy all existing properties
                    'geometry_type': 'hull',
                    'hull_type': 'buffered_convex'
                }
            }
            hull_features.append(hull_feature)
            logger.info(f"Successfully generated hull for district {district}")
        else:
            logger.warning(f"Failed to generate hull for district {district}")
    
    # Create output GeoJSON
    hull_geojson = {
        'type': 'FeatureCollection',
        'features': hull_features,
        'metadata': {
            **point_data['metadata'],  # Copy existing metadata
            'processing': {
                'method': 'buffered_convex_hull',
                'buffer_distance': 0.001,
                'description': 'Smoothed convex hulls generated from postal district points'
            }
        }
    }
    
    # Save the output
    with open(output_path, 'w') as f:
        json.dump(hull_geojson, f, indent=2)
    
    logger.info(f"Saved {len(hull_features)} district hulls to {output_path}")

if __name__ == '__main__':
    process_districts() 