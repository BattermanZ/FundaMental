import React, { useEffect, useState } from 'react';
import { MapContainer, TileLayer, GeoJSON, Tooltip } from 'react-leaflet';
import 'leaflet/dist/leaflet.css';
import { Property } from '../types/property';
import * as d3 from 'd3';
import { Box, Typography, Paper, ButtonGroup, Button } from '@mui/material';

type MapView = 'price' | 'price_per_sqm' | 'density';

interface PriceHeatmapProps {
    properties: Property[];
    metric?: 'price' | 'price_per_sqm' | 'density';  // Optional prop to set initial view
}

interface DistrictData {
    district: string;
    avg_price: number;
    median_price: number;
    avg_price_per_sqm: number;
    median_price_per_sqm: number;
    count: number;
}

// Legend component
const Legend: React.FC<{ min: number; max: number; metric: string }> = ({ min, max, metric }) => {
    const steps = 5; // Number of color steps
    const values = Array.from({ length: steps }, (_, i) => 
        min + (i * (max - min) / (steps - 1))
    );

    const colorScale = d3.scaleSequential()
        .domain([min, max])
        .interpolator((t) => d3.interpolateRdYlGn(1 - t)); // Reverse the scale to get green-to-red

    const formatValue = (value: number) => {
        if (metric === 'price') {
            return `€${(value/1000).toFixed(0)}k`;
        }
        return `€${value.toFixed(0)}/m²`;
    };

    return (
        <Box
            sx={{
                position: 'absolute',
                bottom: '20px',
                right: '20px',
                backgroundColor: 'white',
                padding: '10px',
                borderRadius: '4px',
                boxShadow: '0 0 10px rgba(0,0,0,0.2)',
                zIndex: 1000,
                minWidth: '200px',
            }}
        >
            <Typography variant="subtitle2" gutterBottom>
                {metric === 'price' ? 'Price Range' : 'Price per m² Range'}
            </Typography>
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
                {values.reverse().map((value, i) => (
                    <Box key={i} sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <Box
                            sx={{
                                width: '30px',
                                height: '20px',
                                backgroundColor: colorScale(value),
                                border: '1px solid rgba(0,0,0,0.1)',
                            }}
                        />
                        <Typography variant="caption">
                            {formatValue(value)}
                        </Typography>
                    </Box>
                ))}
            </Box>
        </Box>
    );
};

// Density Legend component
const DensityLegend: React.FC<{ min: number; max: number }> = ({ min, max }) => {
    const steps = 5;
    const values = Array.from({ length: steps }, (_, i) => 
        Math.round(min + (i * (max - min) / (steps - 1)))
    );

    const colorScale = d3.scaleSequential()
        .domain([min, max])
        .interpolator(d3.interpolateBlues);

    return (
        <Box
            sx={{
                position: 'absolute',
                bottom: '20px',
                right: '20px',
                backgroundColor: 'white',
                padding: '10px',
                borderRadius: '4px',
                boxShadow: '0 0 10px rgba(0,0,0,0.2)',
                zIndex: 1000,
                minWidth: '200px',
            }}
        >
            <Typography variant="subtitle2" gutterBottom>
                Properties per District
            </Typography>
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
                {values.reverse().map((value, i) => (
                    <Box key={i} sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <Box
                            sx={{
                                width: '30px',
                                height: '20px',
                                backgroundColor: colorScale(value),
                                border: '1px solid rgba(0,0,0,0.1)',
                            }}
                        />
                        <Typography variant="caption">
                            {value} properties
                        </Typography>
                    </Box>
                ))}
            </Box>
        </Box>
    );
};

const PriceHeatmap: React.FC<PriceHeatmapProps> = ({ properties, metric = 'price' }) => {
    const [currentView, setCurrentView] = useState<MapView>(metric);
    const [districtData, setDistrictData] = useState<Map<string, DistrictData>>(new Map());
    const [geoJsonData, setGeoJsonData] = useState<any>(null);
    const [valueRange, setValueRange] = useState<{ min: number; max: number }>({ min: 0, max: 0 });
    const [densityData, setDensityData] = useState<Map<string, number>>(new Map());
    const [densityRange, setDensityRange] = useState<{ min: number; max: number }>({ min: 0, max: 0 });

    useEffect(() => {
        fetch('/district_hulls.geojson')
            .then(response => response.json())
            .then(data => setGeoJsonData(data))
            .catch(error => console.error('Error loading district hulls:', error));
    }, []);

    useEffect(() => {
        // Calculate price statistics for each district
        const districtGroups = d3.group(
            properties.filter(p => p.price && p.living_area),
            d => d.postal_code.substring(0, 4)
        );

        const newDistrictData = new Map<string, DistrictData>();
        const priceValues: number[] = [];
        const pricePerSqmValues: number[] = [];
        
        districtGroups.forEach((group, district) => {
            const priceValues = group.map(p => p.price);
            const pricePerSqmValues = group.map(p => p.price / (p.living_area || 1));
            
            newDistrictData.set(district, {
                district,
                avg_price: d3.mean(priceValues) || 0,
                median_price: d3.median(priceValues) || 0,
                avg_price_per_sqm: d3.mean(pricePerSqmValues) || 0,
                median_price_per_sqm: d3.median(pricePerSqmValues) || 0,
                count: group.length
            });
        });

        // Calculate density data
        const counts = new Map<string, number>();
        properties.forEach(p => {
            const district = p.postal_code.substring(0, 4);
            counts.set(district, (counts.get(district) || 0) + 1);
        });
        const countValues = Array.from(counts.values());

        setDistrictData(newDistrictData);
        setDensityData(counts);
        setDensityRange({
            min: Math.min(...countValues),
            max: Math.max(...countValues)
        });
    }, [properties]);

    useEffect(() => {
        // Update value range based on current view
        if (districtData.size > 0) {
            const values = Array.from(districtData.values()).map(d => 
                currentView === 'price' ? d.avg_price : 
                currentView === 'price_per_sqm' ? d.avg_price_per_sqm : 0
            );
            setValueRange({
                min: d3.min(values) || 0,
                max: d3.max(values) || 0
            });
        }
    }, [currentView, districtData]);

    if (!geoJsonData) {
        return (
            <Box display="flex" justifyContent="center" alignItems="center" height="400px">
                <Typography>Loading map data...</Typography>
            </Box>
        );
    }

    const getColor = (district: string) => {
        if (currentView === 'density') {
            const count = densityData.get(district) || 0;
            if (count === 0) return '#cccccc';

            const colorScale = d3.scaleSequential()
                .domain([densityRange.min, densityRange.max])
                .interpolator(d3.interpolateBlues);

            return colorScale(count);
        } else {
            const data = districtData.get(district);
            if (!data) return '#cccccc';

            const value = currentView === 'price' ? data.avg_price : data.avg_price_per_sqm;
            const colorScale = d3.scaleSequential()
                .domain([valueRange.min, valueRange.max])
                .interpolator((t) => d3.interpolateRdYlGn(1 - t));

            return colorScale(value);
        }
    };

    const style = (feature: any) => ({
        fillColor: getColor(feature.properties.district),
        weight: 2,
        opacity: 1,
        color: 'white',
        fillOpacity: 0.7
    });

    const formatValue = (value: number, type: 'price' | 'price_per_sqm' | 'count') => {
        if (type === 'price') return `€${(value/1000).toFixed(0)}k`;
        if (type === 'price_per_sqm') return `€${value.toFixed(0)}/m²`;
        return `${value} properties`;
    };

    const onEachFeature = (feature: any, layer: any) => {
        const district = feature.properties.district;
        const data = districtData.get(district);
        const count = densityData.get(district) || 0;

        if (currentView === 'density') {
            layer.bindTooltip(`
                <strong>District: ${district}</strong><br/>
                Number of properties: ${count}
            `);
        } else if (data) {
            layer.bindTooltip(`
                <strong>District: ${district}</strong><br/>
                Average ${currentView === 'price' ? 'Price' : 'Price/m²'}: 
                ${formatValue(currentView === 'price' ? data.avg_price : data.avg_price_per_sqm, currentView)}<br/>
                Median ${currentView === 'price' ? 'Price' : 'Price/m²'}: 
                ${formatValue(currentView === 'price' ? data.median_price : data.median_price_per_sqm, currentView)}<br/>
                Number of properties: ${data.count}
            `);
        }
    };

    const getViewTitle = () => {
        switch (currentView) {
            case 'price': return 'Price by District';
            case 'price_per_sqm': return 'Price per Square Meter by District';
            case 'density': return 'Property Density by District';
        }
    };

    return (
        <Paper sx={{ p: 3 }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                <Typography variant="h6">
                    {getViewTitle()}
                </Typography>
                <ButtonGroup variant="contained" aria-label="map view selection">
                    <Button 
                        onClick={() => setCurrentView('price')}
                        variant={currentView === 'price' ? 'contained' : 'outlined'}
                    >
                        Price
                    </Button>
                    <Button 
                        onClick={() => setCurrentView('price_per_sqm')}
                        variant={currentView === 'price_per_sqm' ? 'contained' : 'outlined'}
                    >
                        Price/m²
                    </Button>
                    <Button 
                        onClick={() => setCurrentView('density')}
                        variant={currentView === 'density' ? 'contained' : 'outlined'}
                    >
                        Density
                    </Button>
                </ButtonGroup>
            </Box>
            <Box sx={{ height: 500, width: '100%', position: 'relative' }}>
                <MapContainer
                    center={[52.3676, 4.9041]}
                    zoom={13}
                    style={{ height: '100%', width: '100%' }}
                >
                    <TileLayer
                        url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
                        attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
                    />
                    <GeoJSON
                        data={geoJsonData}
                        style={style}
                        onEachFeature={onEachFeature}
                    />
                </MapContainer>
                {currentView === 'density' ? (
                    <DensityLegend min={densityRange.min} max={densityRange.max} />
                ) : (
                    <Legend min={valueRange.min} max={valueRange.max} metric={currentView} />
                )}
            </Box>
        </Paper>
    );
};

export default PriceHeatmap; 