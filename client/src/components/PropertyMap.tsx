import React, { useEffect, useState, useCallback } from 'react';
import { MapContainer, TileLayer, Marker, Popup, useMap } from 'react-leaflet';
import 'leaflet/dist/leaflet.css';
import 'leaflet.markercluster/dist/MarkerCluster.css';
import 'leaflet.markercluster/dist/MarkerCluster.Default.css';
import 'leaflet.heat';
import { Property, DateRange } from '../types/property';
import { api } from '../services/api';
import { Icon, LatLngTuple } from 'leaflet';
import { CircularProgress, Typography, Box, Button, FormControl, InputLabel, Select, MenuItem, Slider, Grid } from '@mui/material';
import MarkerClusterGroup from 'react-leaflet-cluster';
import { GeoJSON, Tooltip } from 'react-leaflet';
import { Feature, Polygon } from 'geojson';
import * as d3 from 'd3';

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

interface PropertyMapProps {
    dateRange: DateRange;
}

const PropertyMap: React.FC<PropertyMapProps> = ({ dateRange }) => {
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
            setLoading(true);
            const data = await api.getAllProperties(dateRange);
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
    }, [filters, dateRange]);

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
                        <PriceLayer properties={filteredProperties} />
                    )}
                </MapContainer>
            )}
        </Box>
    );
};

const PriceLayer: React.FC<{ properties: Property[] }> = ({ properties }) => {
    const map = useMap();
    const [priceRange, setPriceRange] = useState<{ min: number; max: number } | null>(null);
    const [voronoiLayer, setVoronoiLayer] = useState<any>(null);

    useEffect(() => {
        if (!properties.length) return;

        // Calculate price range using all properties with valid data
        const validPrices = properties
            .filter(p => p.price && p.living_area && p.living_area > 0)
            .map(p => p.price / p.living_area);

        if (validPrices.length === 0) return;

        const sortedPrices = [...validPrices].sort((a, b) => a - b);
        const p10 = sortedPrices[Math.floor(sortedPrices.length * 0.1)];
        const p90 = sortedPrices[Math.floor(sortedPrices.length * 0.9)];

        setPriceRange({ min: p10, max: p90 });

        // Remove existing layer
        if (voronoiLayer) {
            voronoiLayer.remove();
        }

        // Function to get color based on price
        const getColor = (value: number | null) => {
            if (!value || value <= 0 || !p10 || !p90) return 'transparent';
            
            // Normalize the value between p10 and p90
            const normalized = (value - p10) / (p90 - p10);
            
            // Ensure the normalized value is between 0 and 1
            const capped = Math.min(Math.max(normalized, 0), 1);
            
            // Color scale from green to red
            if (capped <= 0.2) return '#00ff00';  // Green
            if (capped <= 0.4) return '#80ff00';  // Light green
            if (capped <= 0.6) return '#ffff00';  // Yellow
            if (capped <= 0.8) return '#ff8000';  // Orange
            return '#ff0000';                      // Red
        };

        // Create a new SVG layer
        const layer = (window as any).L.svg().addTo(map);
        const svg = d3.select(layer._container);

        // Function to update the Voronoi diagram
        const updateVoronoi = () => {
            // Clear existing paths
            svg.selectAll('path').remove();

            // Get current bounds and zoom level
            const bounds = map.getBounds();
            const zoom = map.getZoom();
            const width = map.getSize().x;
            const height = map.getSize().y;

            // Filter properties within current bounds and add buffer
            const bufferSize = 0.1;
            const extendedBounds = bounds.pad(bufferSize);
            const visibleProperties = properties.filter(p => {
                if (!p.latitude || !p.longitude || !p.price || !p.living_area || p.living_area <= 0) return false;
                return extendedBounds.contains([p.latitude, p.longitude]);
            });

            // Cluster points based on zoom level
            let points;
            if (zoom <= 12) {
                // Group by first 4 digits of postal code for zoomed out view
                const clusters = visibleProperties.reduce((acc: { [key: string]: any }, prop) => {
                    const postal = prop.postal_code.substring(0, 4);
                    if (!acc[postal]) {
                        acc[postal] = {
                            totalPrice: 0,
                            totalArea: 0,
                            count: 0,
                            lat: 0,
                            lng: 0
                        };
                    }
                    acc[postal].totalPrice += prop.price;
                    acc[postal].totalArea += prop.living_area;
                    acc[postal].lat += prop.latitude;
                    acc[postal].lng += prop.longitude;
                    acc[postal].count += 1;
                    return acc;
                }, {});

                points = Object.entries(clusters)
                    .filter(([_, data]) => data.count > 0 && data.totalArea > 0)
                    .map(([postal, data]) => ({
                        lat: data.lat / data.count,
                        lng: data.lng / data.count,
                        pricePerSqm: data.totalPrice / data.totalArea,
                        count: data.count,
                        postal
                    }));
            } else {
                points = visibleProperties.map(prop => ({
                    lat: prop.latitude!,
                    lng: prop.longitude!,
                    pricePerSqm: prop.price / prop.living_area!,
                    count: 1,
                    postal: prop.postal_code
                }));
            }

            if (points.length === 0) return;

            // Add boundary points
            const boundaryPoints = [
                { lat: bounds.getNorth(), lng: bounds.getWest(), pricePerSqm: null, count: 0, postal: '' },
                { lat: bounds.getNorth(), lng: bounds.getEast(), pricePerSqm: null, count: 0, postal: '' },
                { lat: bounds.getSouth(), lng: bounds.getWest(), pricePerSqm: null, count: 0, postal: '' },
                { lat: bounds.getSouth(), lng: bounds.getEast(), pricePerSqm: null, count: 0, postal: '' }
            ];
            points = [...points, ...boundaryPoints];

            // Create and draw Voronoi diagram
            const voronoi = d3.Delaunay
                .from(points, d => map.latLngToLayerPoint([d.lat, d.lng]).x, d => map.latLngToLayerPoint([d.lat, d.lng]).y)
                .voronoi([0, 0, width, height]);

            // Draw polygons
            svg.selectAll('path')
                .data(points)
                .enter()
                .append('path')
                .attr('d', (d, i) => voronoi.renderCell(i))
                .attr('fill', d => getColor(d.pricePerSqm))
                .attr('fill-opacity', 0.5)
                .attr('stroke', 'white')
                .attr('stroke-width', 1)
                .attr('stroke-opacity', 0.8)
                .style('pointer-events', d => d.pricePerSqm === null ? 'none' : 'auto')
                .on('mouseover', (event, d) => {
                    if (d.pricePerSqm === null) return;
                    const tooltip = (window as any).L.tooltip({
                        permanent: false,
                        direction: 'top',
                        className: 'price-tooltip'
                    })
                    .setLatLng([d.lat, d.lng])
                    .setContent(`
                        <strong>Postal Area: ${d.postal}</strong><br/>
                        Average Price/m²: €${Math.round(d.pricePerSqm).toLocaleString()}<br/>
                        Properties: ${d.count}
                    `)
                    .addTo(map);
                    
                    (event.target as any).tooltip = tooltip;
                })
                .on('mouseout', (event) => {
                    const tooltip = (event.target as any).tooltip;
                    if (tooltip) {
                        tooltip.remove();
                        (event.target as any).tooltip = null;
                    }
                });
        };

        // Initial update
        updateVoronoi();

        // Update on map movement and zoom
        map.on('moveend', updateVoronoi);
        map.on('zoomend', updateVoronoi);
        map.on('move', updateVoronoi); // Add update during pan
        
        setVoronoiLayer(layer);

        return () => {
            map.off('moveend', updateVoronoi);
            map.off('zoomend', updateVoronoi);
            map.off('move', updateVoronoi);
            if (voronoiLayer) {
                voronoiLayer.remove();
            }
        };
    }, [map, properties]);

    return priceRange ? <PriceLegend min={priceRange.min} max={priceRange.max} /> : null;
};

const PriceLegend: React.FC<{ min: number; max: number }> = ({ min, max }) => {
    const steps = [0.0, 0.2, 0.4, 0.6, 0.8, 1.0];
    const colors = ['#00ff00', '#80ff00', '#ffff00', '#ff8000', '#ff0000'];
    const values = steps.map(step => Math.round(min + (max - min) * step));

    return (
        <div style={{
            position: 'absolute',
            bottom: '20px',
            right: '20px',
            backgroundColor: 'white',
            padding: '10px',
            borderRadius: '5px',
            boxShadow: '0 0 10px rgba(0,0,0,0.2)',
            zIndex: 1000
        }}>
            <div style={{ marginBottom: '5px', fontWeight: 'bold' }}>Average Price per m² by Area</div>
            {colors.map((color, i) => (
                <div key={i} style={{ display: 'flex', alignItems: 'center', margin: '2px 0' }}>
                    <div style={{
                        width: '20px',
                        height: '20px',
                        backgroundColor: color,
                        marginRight: '5px'
                    }} />
                    <span>€{values[i].toLocaleString()} - €{values[i + 1].toLocaleString()}</span>
                </div>
            ))}
        </div>
    );
};

export default PropertyMap; 