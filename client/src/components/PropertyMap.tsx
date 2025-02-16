import React, { useEffect, useState, useCallback } from 'react';
import { MapContainer, TileLayer, Marker, Popup, useMap } from 'react-leaflet';
import 'leaflet/dist/leaflet.css';
import 'leaflet.markercluster/dist/MarkerCluster.css';
import 'leaflet.markercluster/dist/MarkerCluster.Default.css';
import 'leaflet.heat';
import { Property } from '../types/property';
import { api } from '../services/api';
import { Icon, LatLngTuple } from 'leaflet';
import { CircularProgress, Typography, Box, Button, FormControl, InputLabel, Select, MenuItem, Slider, Grid } from '@mui/material';
import MarkerClusterGroup from 'react-leaflet-cluster';

// Fix for default marker icon
delete (Icon.Default.prototype as any)._getIconUrl;
Icon.Default.mergeOptions({
    iconRetinaUrl: require('leaflet/dist/images/marker-icon-2x.png'),
    iconUrl: require('leaflet/dist/images/marker-icon.png'),
    shadowUrl: require('leaflet/dist/images/marker-shadow.png'),
});

const AMSTERDAM_CENTER: LatLngTuple = [52.3676, 4.9041];

interface FilterOptions {
    priceRange: [number, number];
    sizeRange: [number, number];
    propertyType: string;
    status: string;
}

const PropertyMap: React.FC = () => {
    const [properties, setProperties] = useState<Property[]>([]);
    const [filteredProperties, setFilteredProperties] = useState<Property[]>([]);
    const [loading, setLoading] = useState(true);
    const [geocoding, setGeocoding] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [viewMode, setViewMode] = useState<'markers' | 'heatmap'>('markers');
    const [filters, setFilters] = useState<FilterOptions>({
        priceRange: [0, 2000000],
        sizeRange: [0, 200],
        propertyType: 'all',
        status: 'all'
    });

    const fetchProperties = useCallback(async () => {
        try {
            const data = await api.getAllProperties();
            const validProperties = data.filter(
                (property) => property.latitude != null && 
                            property.longitude != null
            );
            console.log(`Showing ${validProperties.length} properties with coordinates out of ${data.length} total`);
            setProperties(validProperties);
            applyFilters(validProperties, filters);
            setError(null);
        } catch (error) {
            console.error('Failed to fetch properties:', error);
            setError('Failed to load properties. Please try again later.');
        } finally {
            setLoading(false);
        }
    }, [filters]);

    useEffect(() => {
        fetchProperties();
    }, [fetchProperties]);

    const applyFilters = (props: Property[], filterOptions: FilterOptions) => {
        const filtered = props.filter(property => {
            const matchesPrice = property.price >= filterOptions.priceRange[0] && 
                               property.price <= filterOptions.priceRange[1];
            const matchesSize = !property.living_area || 
                              (property.living_area >= filterOptions.sizeRange[0] && 
                               property.living_area <= filterOptions.sizeRange[1]);
            const matchesType = filterOptions.propertyType === 'all' || 
                              property.property_type === filterOptions.propertyType;
            const matchesStatus = filterOptions.status === 'all' || 
                                property.status === filterOptions.status;
            
            return matchesPrice && matchesSize && matchesType && matchesStatus;
        });
        setFilteredProperties(filtered);
    };

    useEffect(() => {
        applyFilters(properties, filters);
    }, [filters, properties]);

    const handleUpdateCoordinates = async () => {
        try {
            setGeocoding(true);
            await api.updateCoordinates();
            await fetchProperties();
        } catch (error) {
            console.error('Failed to update coordinates:', error);
            setError('Failed to update coordinates. Please try again later.');
        } finally {
            setGeocoding(false);
        }
    };

    const formatPrice = (price: number) => 
        `€${price.toLocaleString(undefined, { maximumFractionDigits: 0 })}`;

    const calculatePricePerSqm = (price: number, area: number | null) => {
        if (!area) return null;
        return formatPrice(Math.round(price / area));
    };

    if (loading) {
        return (
            <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '500px' }}>
                <CircularProgress />
            </Box>
        );
    }

    if (error) {
        return (
            <Box sx={{ display: 'flex', flexDirection: 'column', justifyContent: 'center', alignItems: 'center', height: '500px' }}>
                <Typography color="error" gutterBottom>{error}</Typography>
                <Button variant="contained" color="primary" onClick={fetchProperties}>
                    Retry
                </Button>
            </Box>
        );
    }

    return (
        <Box>
            <Box sx={{ mb: 2 }}>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                    <Typography variant="h6">
                        Showing {filteredProperties.length} properties with coordinates
                    </Typography>
                    <Box>
                        <Button 
                            variant="contained" 
                            color="primary" 
                            onClick={handleUpdateCoordinates}
                            disabled={geocoding}
                            sx={{ mr: 1 }}
                        >
                            {geocoding ? 'Updating Coordinates...' : 'Update Coordinates'}
                        </Button>
                        <Button
                            variant="outlined"
                            color="primary"
                            onClick={() => setViewMode(viewMode === 'markers' ? 'heatmap' : 'markers')}
                        >
                            {viewMode === 'markers' ? 'Switch to Heatmap' : 'Switch to Markers'}
                        </Button>
                    </Box>
                </Box>

                <Grid container spacing={2}>
                    <Grid item xs={12} md={3}>
                        <FormControl fullWidth>
                            <InputLabel>Property Type</InputLabel>
                            <Select
                                value={filters.propertyType}
                                label="Property Type"
                                onChange={(e) => setFilters({ ...filters, propertyType: e.target.value as string })}
                            >
                                <MenuItem value="all">All Types</MenuItem>
                                <MenuItem value="appartement">Apartment</MenuItem>
                                <MenuItem value="house">House</MenuItem>
                            </Select>
                        </FormControl>
                    </Grid>

                    <Grid item xs={12} md={3}>
                        <FormControl fullWidth>
                            <InputLabel>Status</InputLabel>
                            <Select
                                value={filters.status}
                                label="Status"
                                onChange={(e) => setFilters({ ...filters, status: e.target.value as string })}
                            >
                                <MenuItem value="all">All Status</MenuItem>
                                <MenuItem value="active">Active</MenuItem>
                                <MenuItem value="sold">Sold</MenuItem>
                            </Select>
                        </FormControl>
                    </Grid>

                    <Grid item xs={12} md={3}>
                        <Box>
                            <Typography gutterBottom>Price Range</Typography>
                            <Slider
                                value={filters.priceRange}
                                onChange={(_, newValue) => setFilters({ ...filters, priceRange: newValue as [number, number] })}
                                valueLabelDisplay="auto"
                                min={0}
                                max={2000000}
                                step={50000}
                                valueLabelFormat={value => `€${(value/1000)}k`}
                            />
                        </Box>
                    </Grid>

                    <Grid item xs={12} md={3}>
                        <Box>
                            <Typography gutterBottom>Size Range (m²)</Typography>
                            <Slider
                                value={filters.sizeRange}
                                onChange={(_, newValue) => setFilters({ ...filters, sizeRange: newValue as [number, number] })}
                                valueLabelDisplay="auto"
                                min={0}
                                max={200}
                                step={10}
                            />
                        </Box>
                    </Grid>
                </Grid>
            </Box>
            
            {filteredProperties.length === 0 ? (
                <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '500px' }}>
                    <Typography>No properties match the current filters</Typography>
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
                    {viewMode === 'markers' ? (
                        <MarkerClusterGroup
                            chunkedLoading
                            maxClusterRadius={60}
                        >
                            {filteredProperties.map((property) => (
                                <Marker
                                    key={property.id}
                                    position={[property.latitude!, property.longitude!] as LatLngTuple}
                                >
                                    <Popup>
                                        <div>
                                            <h3>{property.street}</h3>
                                            <p><strong>Price:</strong> {formatPrice(property.price)}</p>
                                            {property.living_area && (
                                                <>
                                                    <p><strong>Area:</strong> {property.living_area}m²</p>
                                                    <p><strong>Price/m²:</strong> {calculatePricePerSqm(property.price, property.living_area)}</p>
                                                </>
                                            )}
                                            {property.property_type && (
                                                <p><strong>Type:</strong> {property.property_type}</p>
                                            )}
                                            {property.year_built && (
                                                <p><strong>Year built:</strong> {property.year_built}</p>
                                            )}
                                            <p><strong>Status:</strong> {property.status}</p>
                                            {property.selling_date && (
                                                <p><strong>Sold on:</strong> {new Date(property.selling_date).toLocaleDateString()}</p>
                                            )}
                                            <a href={property.url} target="_blank" rel="noopener noreferrer">
                                                View on Funda
                                            </a>
                                        </div>
                                    </Popup>
                                </Marker>
                            ))}
                        </MarkerClusterGroup>
                    ) : (
                        <HeatmapLayer properties={filteredProperties} />
                    )}
                </MapContainer>
            )}
        </Box>
    );
};

// Heatmap layer component
const HeatmapLayer: React.FC<{ properties: Property[] }> = ({ properties }) => {
    const map = useMap();

    useEffect(() => {
        if (!properties.length) return;

        const points = properties.map(p => [
            p.latitude!,
            p.longitude!,
            p.price / 1000000 // Normalize price for heat intensity
        ]);

        const heat = (window as any).L.heatLayer(points, {
            radius: 25,
            blur: 15,
            maxZoom: 10,
            max: 2.0, // Maximum price in millions for intensity scaling
            gradient: {0.4: 'blue', 0.65: 'lime', 0.85: 'yellow', 1: 'red'}
        });

        heat.addTo(map);
        return () => {
            map.removeLayer(heat);
        };
    }, [map, properties]);

    return null;
};

export default PropertyMap; 