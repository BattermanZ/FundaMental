import React, { useEffect, useState } from 'react';
import { MapContainer, TileLayer, Marker, Popup } from 'react-leaflet';
import 'leaflet/dist/leaflet.css';
import { Property } from '../types/property';
import { api } from '../services/api';
import { Icon } from 'leaflet';

// Fix for default marker icon
delete (Icon.Default.prototype as any)._getIconUrl;
Icon.Default.mergeOptions({
    iconRetinaUrl: require('leaflet/dist/images/marker-icon-2x.png'),
    iconUrl: require('leaflet/dist/images/marker-icon.png'),
    shadowUrl: require('leaflet/dist/images/marker-shadow.png'),
});

const AMSTERDAM_CENTER = [52.3676, 4.9041];

const PropertyMap: React.FC = () => {
    const [properties, setProperties] = useState<Property[]>([]);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        const fetchProperties = async () => {
            try {
                const data = await api.getAllProperties();
                setProperties(data);
            } catch (error) {
                console.error('Failed to fetch properties:', error);
            } finally {
                setLoading(false);
            }
        };

        fetchProperties();
    }, []);

    if (loading) {
        return <div>Loading map...</div>;
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
            {properties.map((property) => (
                <Marker
                    key={property.id}
                    position={[52.3676, 4.9041]} // TODO: Add geocoding to get actual coordinates
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
            ))}
        </MapContainer>
    );
};

export default PropertyMap; 