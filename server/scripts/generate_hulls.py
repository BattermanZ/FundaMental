import json
from pathlib import Path
import logging
import sys
from geometry.hull_generator import generate_district_hull
import os

# Set up logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def process_districts(input_data):
    """
    Process the point data for each postal district and generate hull polygons.
    
    Args:
        input_data: Dictionary containing district points data from Go app
    """
    logger.info("Processing district point data from Go app...")
    
    # Get the project root directory (three levels up from this script)
    project_root = Path(os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))
    output_path = project_root / 'client' / 'public' / 'district_hulls.geojson'
    
    # Process each district
    hull_features = []
    for feature in input_data['features']:
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
            **input_data.get('metadata', {}),  # Copy existing metadata
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
    
    # Return success status to Go app
    print(json.dumps({"status": "success", "hull_count": len(hull_features)}))

if __name__ == '__main__':
    # Read input data from stdin (sent by Go app)
    input_data = json.load(sys.stdin)
    process_districts(input_data) 