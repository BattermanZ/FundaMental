import React, { useEffect, useState } from 'react';
import { MapContainer, TileLayer, Marker, Popup } from 'react-leaflet';
import 'leaflet/dist/leaflet.css';
import { Property } from '../types/property';
import { api } from '../services/api';
import { Icon, LatLngTuple } from 'leaflet';
import { CircularProgress, Typography, Box } from '@material-ui/core';

// Fix for default marker icon
delete (Icon.Default.prototype as any)._getIconUrl;
Icon.Default.mergeOptions({
    iconRetinaUrl: require('leaflet/dist/images/marker-icon-2x.png'),
    iconUrl: require('leaflet/dist/images/marker-icon.png'),
    shadowUrl: require('leaflet/dist/images/marker-shadow.png'),
});

const AMSTERDAM_CENTER: LatLngTuple = [52.3676, 4.9041];

const PropertyMap: React.FC = () => {
    const [properties, setProperties] = useState<Property[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const fetchProperties = async () => {
            try {
                const data = await api.getAllProperties();
                console.log('Fetched properties:', data);
                setProperties(data);
                setError(null);
            } catch (error) {
                console.error('Failed to fetch properties:', error);
                setError('Failed to load properties. Please try again later.');
            } finally {
                setLoading(false);
            }
        };

        fetchProperties();
    }, []);

    if (loading) {
        return (
            <Box display="flex" justifyContent="center" alignItems="center" height="500px">
                <CircularProgress />
            </Box>
        );
    }

    if (error) {
        return (
            <Box display="flex" justifyContent="center" alignItems="center" height="500px">
                <Typography color="error">{error}</Typography>
            </Box>
        );
    }

    if (properties.length === 0) {
        return (
            <Box display="flex" justifyContent="center" alignItems="center" height="500px">
                <Typography>No properties found</Typography>
            </Box>
        );
    }

    return (
        <MapContainer
            center={AMSTERDAM_CENTER}
            zoom={13}
            style={{ height: '500px', width: '100%' }}
        >
            <TileLayer
                url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
                attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
            />
            {properties.map((property) => {
                // For now, use a random offset from Amsterdam center for visualization
                const randomLat = AMSTERDAM_CENTER[0] + (Math.random() - 0.5) * 0.02;
                const randomLng = AMSTERDAM_CENTER[1] + (Math.random() - 0.5) * 0.02;
                
                return (
                    <Marker
                        key={property.id}
                        position={[randomLat, randomLng] as LatLngTuple}
                    >
                        <Popup>
                            <div>
                                <h3>{property.street}</h3>
                                <p>Price: €{property.price.toLocaleString()}</p>
                                <p>Area: {property.living_area}m²</p>
                                <p>Type: {property.property_type}</p>
                                <p>Status: {property.status}</p>
                                <a href={property.url} target="_blank" rel="noopener noreferrer">
                                    View on Funda
                                </a>
                            </div>
                        </Popup>
                    </Marker>
                );
            })}
        </MapContainer>
    );
};

export default PropertyMap; 