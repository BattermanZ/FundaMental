import React, { useEffect, useState } from 'react';
import { MapContainer, TileLayer, Marker, Popup } from 'react-leaflet';
import 'leaflet/dist/leaflet.css';
import { Property } from '../types/property';
import { api } from '../services/api';
import { Icon, LatLngTuple } from 'leaflet';
import { CircularProgress, Typography, Box, Button } from '@material-ui/core';

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
    const [geocoding, setGeocoding] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const fetchProperties = async () => {
        try {
            const data = await api.getAllProperties();
            // Filter properties to only include those with valid coordinates
            const validProperties = data.filter(
                (property) => property.latitude != null && 
                            property.longitude != null
            );
            console.log(`Showing ${validProperties.length} properties with coordinates out of ${data.length} total`);
            setProperties(validProperties);
            setError(null);
        } catch (error) {
            console.error('Failed to fetch properties:', error);
            setError('Failed to load properties. Please try again later.');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchProperties();
    }, []);

    const handleUpdateCoordinates = async () => {
        try {
            setGeocoding(true);
            await api.updateCoordinates();
            // Refetch properties after geocoding
            await fetchProperties();
        } catch (error) {
            console.error('Failed to update coordinates:', error);
            setError('Failed to update coordinates. Please try again later.');
        } finally {
            setGeocoding(false);
        }
    };

    if (loading) {
        return (
            <Box display="flex" justifyContent="center" alignItems="center" height="500px">
                <CircularProgress />
            </Box>
        );
    }

    if (error) {
        return (
            <Box display="flex" flexDirection="column" justifyContent="center" alignItems="center" height="500px">
                <Typography color="error" gutterBottom>{error}</Typography>
                <Button variant="contained" color="primary" onClick={fetchProperties}>
                    Retry
                </Button>
            </Box>
        );
    }

    return (
        <Box>
            <Box mb={2} display="flex" justifyContent="space-between" alignItems="center">
                <Typography variant="h6">
                    Showing {properties.length} properties with coordinates
                </Typography>
                <Button 
                    variant="contained" 
                    color="primary" 
                    onClick={handleUpdateCoordinates}
                    disabled={geocoding}
                >
                    {geocoding ? 'Updating Coordinates...' : 'Update Coordinates'}
                </Button>
            </Box>
            
            {properties.length === 0 ? (
                <Box display="flex" justifyContent="center" alignItems="center" height="500px">
                    <Typography>No properties with coordinates found</Typography>
                </Box>
            ) : (
                <MapContainer
                    center={AMSTERDAM_CENTER}
                    zoom={13}
                    style={{ height: '500px', width: '100%' }}
                >
                    <TileLayer
                        url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
                        attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
                    />
                    {properties.map((property) => (
                        <Marker
                            key={property.id}
                            position={[property.latitude!, property.longitude!] as LatLngTuple}
                        >
                            <Popup>
                                <div>
                                    <h3>{property.street}</h3>
                                    <p>Price: €{property.price.toLocaleString()}</p>
                                    {property.living_area && (
                                        <p>Area: {property.living_area}m²</p>
                                    )}
                                    {property.property_type && (
                                        <p>Type: {property.property_type}</p>
                                    )}
                                    <p>Status: {property.status}</p>
                                    <a href={property.url} target="_blank" rel="noopener noreferrer">
                                        View on Funda
                                    </a>
                                </div>
                            </Popup>
                        </Marker>
                    ))}
                </MapContainer>
            )}
        </Box>
    );
};

export default PropertyMap; 