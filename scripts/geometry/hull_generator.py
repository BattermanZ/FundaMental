import numpy as np
from shapely.geometry import MultiPoint, Polygon
import logging

# Set up logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def generate_district_hull(points, buffer_distance=0.001):
    """
    Generate a hull for a postal district from its points.
    
    Args:
        points: List of [lon, lat] coordinates
        buffer_distance: Distance to buffer the hull (in degrees, default: 0.001)
                      This creates a smoother, more natural looking boundary
    
    Returns:
        GeoJSON-compatible dictionary representing the hull polygon
    """
    try:
        # Convert points to numpy array
        points_array = np.array(points)
        
        # Generate the convex hull
        hull = MultiPoint(points_array).convex_hull
        
        # Buffer the hull to smooth it
        smoothed_hull = hull.buffer(buffer_distance)
        
        # Convert coordinates to GeoJSON format
        coords = [[[float(x), float(y)] for x, y in smoothed_hull.exterior.coords]]
        
        return {
            "type": "Polygon",
            "coordinates": coords
        }
    except Exception as e:
        logger.error(f"Error generating hull: {e}")
        return None 